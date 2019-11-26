# clist

clist aims to be an RFC-compliant mailing list manager.
When users send an email to a list, clist delivers that email to other subscribers of that list.
Users can manage their subscriptions by sending commands to clist over email.
clist can manage multiple lists, each with its own set of subscribers.
clist is intended to be used by a single organization and will provide a sane set of defaults and a simple configuration.

## Dependencies

clist requires:

* a Go compiler
* [go-sqlite3](https://github.com/mattn/go-sqlite3)
* an encrypted SMTP server to send mail
* an MTA to handle receiving mail

## Installation

To install:

1. clone the repository.
2. run `make clean install` to compile + install clist.

## Configuration

The configuration file for clist is `/etc/clist.ini`.
The configuration tells clist how to connect to its local database
and how to connect to an SMTP server.
It also contains sections for each list that tell clist how each list should behave.

### Basic Configuration

A basic configuration looks like this:

```
log = /var/log/clist
database = /var/db/clist/clist.db
command_address = majordomo@example.com
smtp_hostname = smtp.example.com
smtp_port = 587
smtp_username = AzureDiamond
smtp_password = hunter2
```

**command_address**

The email address that clist will use to accept commands.

**log**

Path to the file that clist will log to.

**database**

The name of the sqlite database where clist will store data.

**smtp_hostname**

Hostname of the SMTP server that clist will use to send outgoing mail.

**smtp_port**

SMTP port number. Usually 25, 587, or 2525 for unencrypted connections, and 465 or 25025 for encrypted transport.

**smtp_username**  
**smtp_password**

Username and password for the SMTP server.

### List Configuration

Each list section looks like this:

```
[list.test]
address = test@example.com
name = "Test"
description = "Test List"
subscribers_only = true
archive = "https://archive.example.com/test"
owner = "Lain Iwakura"
posters = "j3s@example.com,aedric@example.com" # optional
```

**id**

A short identifier for this list.
Used when subscribing or unsubscribing to the list. ([list.test] would mean the identifier is "test")

**address**

Email address of list

**name**

Shortname for mailing list (openbsd-misc, Misc, BurgerTown)

**archive**

An email address. The value for this setting is set as the List-Archive header on outgoing messages. Per RFC 5064, this value described how to access archives for the list.

**owner**

An email address inserted into outgoing mail as the List-Owner header per RFC 5064.

**description**

A description of this list. 
This value is provided in the listing of lists given in response to a request for a list of mailing lists.

**subscribers_only**

If set to true, only subscribers may send messages to the list.
We recommend sending this to true.

**posters,omitempty**

A comma-separated whitelist of approved posters, if you want the list to be restrictive (announce lists, for example)

Using clist
-----------

clist takes input on stdin, and responds appropriately via the SMTP server that has been configured.

getting data to clist's stdin can be as simple as modifying `/etc/mail/aliases`

```
majordomo: | /usr/local/bin/clist -config /etc/clist.ini message
```

clist will monitor the subject line of emails sent to the command address.
To get a list of available commands, send an email with 'help' in the subject line to the command address.

Available commands are:

**help**

Reply with a list of valid commands.

**lists**

Reply with a list of mailing lists.

**subscribe**

The subscribe command must be followed by a valid list Id.
clist will add the address in the From head to the list following a confirmation.

**unsubscribe**

The unsubscribe command must be followed by a valid list Id.
clist will remove the address in the From head to the list.

## Contributing

To help with development, please send patches, ideas, and bug reports to misc@c3f.net
