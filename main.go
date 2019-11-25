package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	"git.cyberia.club/cyberia-services/clist/mail"
	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/ini.v1"
	"io"
	"log"
	"net/mail"
	"net/smtp"
	"os"
	"regexp"
	"strings"
	"time"
)

type Config struct {
	CommandAddress string `ini:"command_address"`
	Log            string `ini:"log"`
	Database       string `ini:"database"`
	SMTPHostname   string `ini:"smtp_hostname"`
	SMTPPort       string `ini:"smtp_port"`
	SMTPUsername   string `ini:"smtp_username"`
	SMTPPassword   string `ini:"smtp_password"`
	Lists          map[string]*List
	Debug          bool
	ConfigFile     string
}

type List struct {
	Name            string `ini:"name"`
	Archive         string `ini:"archive"`
	Owner           string `ini:"owner"`
	Description     string `ini:"description"`
	Id              string
	Address         string   `ini:"address"`
	Hidden          bool     `ini:"hidden"`
	SubscribersOnly bool     `ini:"subscribers_only"`
	Posters         []string `ini:"posters,omitempty"`
	Bcc             []string `ini:"bcc,omitempty"`
}

var gConfig *Config

// Entry point
func main() {
	gConfig = &Config{}

	flag.StringVar(&gConfig.ConfigFile, "config", "", "Load configuration from specified file")
	flag.Parse()

	loadConfig()

	if len(flag.Args()) < 1 {
		fmt.Printf("Error: Command not specified\n")
		os.Exit(1)
	}

	requireLog()

	if flag.Arg(0) == "message" {
		msg, err := parsemail.Parse(bufio.NewReader(os.Stdin))
		if err != nil {
			log.Printf("ERROR_PARSING_MESSAGE Error=%q\n", err.Error())
			os.Exit(0)
		}
		log.Printf("MESSAGE_RECEIVED From=%q To=%q Cc=%q Bcc=%q Subject=%q\n",
			msg.From, msg.To, msg.Cc, msg.Bcc, msg.Subject)
		handleMessage(&msg)
	} else {
		fmt.Printf("Unknown command %s\n", flag.Arg(0))
	}
}

func checkAddress(addrs []*mail.Address, checkAddr string) bool {
	for _, a := range addrs {
		if a.Address == checkAddr {
			return true
		}
	}
	return false
}

// Figure out if this is a command, or a mailing list post
func handleMessage(msg *parsemail.Email) {
	if checkAddress(msg.To, gConfig.CommandAddress) {
		handleCommand(msg)
	} else {
		matchedLists := []*List{}
		for _, l := range gConfig.Lists {
			agg := append(msg.To, msg.Cc...)
			if checkAddress(agg, l.Address) {
				matchedLists = append(matchedLists, l)
				log.Printf("matched list: %q", l.Address)
			}
		}

		if len(matchedLists) == 1 {
			list := matchedLists[0]
			if list.CanPost(msg.From[0]) {
				msg := buildListEmail(msg, list)
				send(msg)
				log.Printf("MESSAGE_SENT ListId=%q",
					list.Id)
			} else {
				handleNotAuthorisedToPost(msg)
			}
		} else {
			log.Printf("LISTS: %q", msg)
			handleNoDestination(msg)
		}
	}
}

func subjectParser(s string) string {
	var subject string

	re := regexp.MustCompile(`[Ll]s|[Ll]ists?`)
	if re.MatchString(s) {
		subject = "lists"
	}

	re = regexp.MustCompile(`[Hh]elp`)
	if re.MatchString(s) {
		subject = "help"
	}

	re = regexp.MustCompile(`[Ss]ubscribe `)
	if re.MatchString(s) {
		subject = "subscribe"
	}

	re = regexp.MustCompile(`[Uu]nsubscribe `)
	if re.MatchString(s) {
		subject = "unsubscribe"
	}

	return subject
}

// Handle the command given by the user
func handleCommand(msg *parsemail.Email) {
	switch subjectParser(msg.Subject) {
	case "lists":
		handleShowLists(msg)
	case "help":
		handleHelp(msg)
	case "subscribe":
		handleSubscribe(msg)
	case "unsubscribe":
		handleUnsubscribe(msg)
	default:
		handleUnknownCommand(msg)
	}
}

// Reply to a message that has nowhere to go
func handleNoDestination(msg *parsemail.Email) {
	var body bytes.Buffer
	fmt.Fprintf(&body, "No mailing lists addressed. Your message has not been delivered.\r\n")
	reply := buildCommandEmail(msg, body)
	send(reply)
	log.Printf("UNKNOWN_DESTINATION From=%q To=%q Cc=%q Bcc=%q", msg.From, msg.To, msg.Cc, msg.Bcc)
}

// Reply that the user isn't authorised to post to the list
func handleNotAuthorisedToPost(msg *parsemail.Email) {
	var body bytes.Buffer
	fmt.Fprintf(&body, "You are not an approved poster for this mailing list. Your message has not been delivered.\r\n")
	reply := buildCommandEmail(msg, body)
	send(reply)
	log.Printf("UNAUTHORISED_POST From=%q To=%q Cc=%q Bcc=%q", msg.From, msg.To, msg.Cc, msg.Bcc)
}

// Reply to an unknown command, giving some help
func handleUnknownCommand(msg *parsemail.Email) {
	var body bytes.Buffer
	fmt.Fprintf(&body,
		"%s is not a valid command.\r\n\r\n"+
			"Valid commands are:\r\n\r\n"+
			commandInfo(),
		msg.Subject)
	reply := buildCommandEmail(msg, body)
	send(reply)
	log.Printf("UNKNOWN_COMMAND From=%q", msg.From)
}

// Reply to a help command with help information
func handleHelp(msg *parsemail.Email) {
	var body bytes.Buffer
	fmt.Fprintf(&body, commandInfo())
	reply := buildCommandEmail(msg, body)
	send(reply)
	log.Printf("HELP_SENT To=%q", reply.To)
}

// Reply to a show mailing lists command with a list of mailing lists
func handleShowLists(msg *parsemail.Email) {
	var body bytes.Buffer
	fmt.Fprintf(&body, "Available mailing lists\r\n")
	fmt.Fprintf(&body, "-----------------------\r\n\r\n")
	for _, list := range gConfig.Lists {
		if !list.Hidden {
			fmt.Fprintf(&body,
				"%s\r\n============\r\n"+
					"%s\r\n\r\n",
				list.Id, list.Description)
		}
	}

	log.Printf("SEND")
	fmt.Fprintf(&body,
		"\r\nTo subscribe to a mailing list, email %s with 'subscribe <list-id>' as the subject.\r\n",
		gConfig.CommandAddress)

	log.Printf("SEND")
	email := buildCommandEmail(msg, body)
	send(email)
	log.Printf("LIST_SENT To=%q", msg.From)
}

// Handle a subscribe command
func handleSubscribe(msg *parsemail.Email) {
	listId := strings.TrimPrefix(msg.Subject, "Subscribe ")
	listId = strings.TrimPrefix(listId, "subscribe ")
	list := lookupList(listId)

	if list == nil {
		handleInvalidRequest(msg, listId)
		os.Exit(0)
	}

	var body bytes.Buffer
	if isSubscribed(msg.From[0], listId) {
		fmt.Fprintf(&body, "You are already subscribed to %s\r\n", listId)
		log.Printf("DUPLICATE_SUBSCRIPTION_REQUEST User=%q List=%q\n", msg.From, listId)
	} else {
		addSubscription(msg.From[0], listId)
		fmt.Fprintf(&body, "You are now subscribed to %s\r\n", listId)
		fmt.Fprintf(&body, "To send a message to this list, send an email to %s\r\n", list.Address)
	}
	reply := buildCommandEmail(msg, body)
	send(reply)
}

// Handle an unsubscribe command
func handleUnsubscribe(msg *parsemail.Email) {
	listId := strings.TrimPrefix(msg.Subject, "Unsubscribe ")
	listId = strings.TrimPrefix(listId, "unsubscribe ")
	list := lookupList(listId)

	if list == nil {
		handleInvalidRequest(msg, listId)
		os.Exit(0)
	}

	var body bytes.Buffer
	if !isSubscribed(msg.From[0], listId) {
		fmt.Fprintf(&body, "You aren't subscribed to %s\r\n", listId)
		log.Printf("DUPLICATE_UNSUBSCRIPTION_REQUEST User=%q List=%q\n", msg.From, listId)
	} else {
		removeSubscription(msg.From[0], listId)
		fmt.Fprintf(&body, "You are now unsubscribed from %s\r\n", listId)
	}
	reply := buildCommandEmail(msg, body)
	send(reply)
}

func handleInvalidRequest(msg *parsemail.Email, listId string) {
	var body bytes.Buffer
	fmt.Fprintf(&body, "Unable to operate against %s, Invalid mailing list ID.\r\n", listId)
	reply := buildCommandEmail(msg, body)
	send(reply)
	log.Printf("INVALID_MAILING_LIST From=%q To=%q Cc=%q Bcc=%q", msg.From, msg.To, msg.Cc, msg.Bcc)
}

func badAddress(recipient string, e *parsemail.Email) bool {
	for _, list := range gConfig.Lists {
		// We are a bad address if we are part of the list
		if recipient == list.Address {
			return true
		}
	}

	// We are a bad address if we are already in to/cc
	for _, a := range append(append(e.To, e.Cc...), e.Bcc...) {
		if recipient == a.Address {
			return true
		}
	}
	return false
}

func buildCommandEmail(e *parsemail.Email, t bytes.Buffer) *parsemail.Email {
	response := parsemail.Email{}
	response.Sender = &mail.Address{"", gConfig.CommandAddress}
	from := &mail.Address{"clist", gConfig.CommandAddress}
	response.From = []*mail.Address{from}
	response.To = e.From
	response.Bcc = e.From
	response.Subject = "Re: " + e.Subject
	header := make(map[string][]string)
	header["Date"] = []string{time.Now().Format("Mon, 2 Jan 2006 15:04:05 -0700")}
	header["Precedence"] = []string{"list"}
	header["List-Help"] = []string{"<mailto:" + gConfig.CommandAddress + "?subject=help>"}
	response.Header = header
	response.Bcc = e.From
	response.TextBody = t.String()
	log.Printf("%q", response)
	return &response
}

func lookupList(l string) *List {
	for _, list := range gConfig.Lists {
		if l == list.Id {
			return list
		}
	}
	return nil
}

func buildListEmail(e *parsemail.Email, l *List) *parsemail.Email {
	// Build recipient list, stripping garbage
	var recipients []*mail.Address
	for _, subscriber := range fetchSubscribers(l.Id) {
		if !badAddress(subscriber, e) {
			recipients = append(recipients, &mail.Address{"", subscriber})
		}
	}

	post := e
	post.Sender = &mail.Address{l.Name, l.Address}
	post.Bcc = recipients
	post.Header["Return-Path"] = []string{"bounce-" + l.Address}
	post.Header["Date"] = e.Header["Date"] // RFC 1123
	post.Header["Precedence"] = []string{"list"}
	post.Header["List-Id"] = []string{"<" + strings.Replace(l.Address, "@", ".", -1) + ">"}
	post.Header["List-Post"] = []string{"<mailto:" + l.Address + ">"}
	post.Header["List-Help"] = []string{"<mailto:" + l.Address + "?subject=help>"}
	post.Header["List-Subscribe"] = []string{"<mailto:" + gConfig.CommandAddress + "?subject=subscribe%20" + l.Id + ">"}
	post.Header["List-Unsubscribe"] = []string{"<mailto:" + gConfig.CommandAddress + "?subject=unsubscribe%20" + l.Id + ">"}
	post.Header["List-Archive"] = []string{"<" + l.Archive + ">"}
	post.Header["List-Owner"] = []string{"<" + l.Owner + ">"}
	return post
}

func send(e *parsemail.Email) {
	// Bcc = recipients
	var recipients []string
	for _, a := range e.Bcc {
		recipients = append(recipients, a.Address)
	}

	auth := smtp.PlainAuth("", gConfig.SMTPUsername, gConfig.SMTPPassword, "mail.c3f.net")
	smtp.SendMail("mail.c3f.net:587", auth, e.Sender.Address, recipients, e.ToBytes())
}

// MAILING LIST LOGIC /////////////////////////////////////////////////////////

// Check if the user is authorised to post to this mailing list
func (list *List) CanPost(from *mail.Address) bool {

	// Is this list restricted to subscribers only?
	if list.SubscribersOnly && !isSubscribed(from, list.Id) {
		return false
	}

	// Is there a whitelist of approved posters?
	if len(list.Posters) > 0 {
		for _, poster := range list.Posters {
			if from.Address == poster {
				return true
			}
		}
		return false
	}

	return true
}

// DATABASE LOGIC /////////////////////////////////////////////////////////////

// Open the database
func openDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", gConfig.Database)

	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS "subscriptions" (
		"list" TEXT,
		"user" TEXT
	);
	`)

	return db, err
}

// Open the database or fail immediately
func requireDB() *sql.DB {
	db, err := openDB()
	if err != nil {
		log.Printf("DATABASE_ERROR Error=%q\n", err.Error())
		os.Exit(1)
	}
	return db
}

// Fetch list of subscribers to a mailing list from database
func fetchSubscribers(listId string) []string {
	db := requireDB()
	rows, err := db.Query("SELECT user FROM subscriptions WHERE list=?", listId)

	if err != nil {
		log.Printf("DATABASE_ERROR Error=%q\n", err.Error())
		os.Exit(0)
	}

	listIds := []string{}
	defer rows.Close()
	for rows.Next() {
		var user string
		rows.Scan(&user)
		listIds = append(listIds, user)
	}

	return listIds
}

// Check if a user is subscribed to a mailing list
func isSubscribed(address *mail.Address, list string) bool {
	db := requireDB()

	exists := false
	err := db.QueryRow("SELECT 1 FROM subscriptions WHERE user=? AND list=?", address.Address, list).Scan(&exists)

	if err == sql.ErrNoRows {
		return false
	} else if err != nil {
		log.Printf("DATABASE_ERROR Error=%q\n", err.Error())
		os.Exit(0)
	}

	return true
}

// Add a subscription to the subscription database
func addSubscription(address *mail.Address, list string) {
	db := requireDB()
	_, err := db.Exec("INSERT INTO subscriptions (user,list) VALUES(?,?)", address.Address, list)
	if err != nil {
		log.Printf("DATABASE_ERROR Error=%q\n", err.Error())
		os.Exit(0)
	}
	log.Printf("SUBSCRIPTION_ADDED Sender=%q List=%q\n", address, list)
}

// Remove a subscription from the subscription database
func removeSubscription(address *mail.Address, list string) {
	db := requireDB()
	_, err := db.Exec("DELETE FROM subscriptions WHERE user=? AND list=?", address.Address, list)
	if err != nil {
		log.Printf("DATABASE_ERROR Error=%q\n", err.Error())
		os.Exit(0)
	}
	log.Printf("SUBSCRIPTION_REMOVED Sender=%q List=%q\n", address, list)
}

// HELPER FUNCTIONS ///////////////////////////////////////////////////////////

// Open the log file for logging
func openLog() error {
	logFile, err := os.OpenFile(gConfig.Log, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	out := io.MultiWriter(logFile, os.Stderr)
	log.SetOutput(out)
	return nil
}

// Open the log, or fail immediately
func requireLog() {
	err := openLog()
	if err != nil {
		log.Printf("LOG_ERROR Error=%q\n", err.Error())
		os.Exit(0)
	}
}

// Load gConfig from the on-disk config file
func loadConfig() {
	var (
		err error
		cfg *ini.File
	)

	if len(gConfig.ConfigFile) > 0 {
		cfg, err = ini.Load(gConfig.ConfigFile)
	} else {
		cfg, err = ini.LooseLoad("clist.ini", "/etc/clist.ini")
	}

	if err != nil {
		log.Printf("CONFIG_ERROR Error=%q\n", err.Error())
		os.Exit(0)
	}

	err = cfg.Section("").MapTo(gConfig)
	if err != nil {
		log.Printf("CONFIG_ERROR Error=%q\n", err.Error())
		os.Exit(0)
	}

	gConfig.Lists = make(map[string]*List)

	for _, section := range cfg.ChildSections("list") {
		list := &List{}
		err = section.MapTo(list)
		if err != nil {
			log.Printf("CONFIG_ERROR Error=%q\n", err.Error())
			os.Exit(0)
		}
		list.Id = strings.TrimPrefix(section.Name(), "list.")
		gConfig.Lists[list.Address] = list
	}
}

// Generate an email-able list of commands
func commandInfo() string {
	return fmt.Sprintf("    help\r\n"+
		"      Information about valid commands\r\n"+
		"\r\n"+
		"    list\r\n"+
		"      Retrieve a list of available mailing lists\r\n"+
		"\r\n"+
		"    subscribe <list-id>\r\n"+
		"      Subscribe to <list-id>\r\n"+
		"\r\n"+
		"    unsubscribe <list-id>\r\n"+
		"      Unsubscribe from <list-id>\r\n"+
		"\r\n"+
		"To send a command, email %s with the command as the subject.\r\n",
		gConfig.CommandAddress)
}
