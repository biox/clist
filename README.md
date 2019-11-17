# clist

clist aims to be an RFC-compliant mailing list manager.
When users send an email to a list, clist delivers that email to other subscribers of that list.
Users can manage their subscriptions by sending commands to clist over email.
clist can manage multiple lists, each with its own set of subscribers.

## Dependencies

clist requires:

* a Go compiler
* [go-sqlite3](https://github.com/mattn/go-sqlite3)
* an SMTP server to send mail
* an MTA to handle receiving mail

## Installation

To install:

1. clone the repository.
2. change to the clist directory and source `scripts/quickdeploy` to build and install clist.

## Configuration

The configuration file for clist is `/etc/clist.ini`.
The configuration tells clist how to connect to its local database
and how to connect to an SMTP server.
It also contains sections for each list that tell clist how each list should behave.

### Basic Configuration

A basic configuration looks like this:

```
command_address = 
log = /path/to/logfile
database = dbname
smtp_hostname = your_smtp_host
smtp_port = 587
smtp_username = AzureDiamond
smtp_password = hunter2
Debug = true | false
ConfigFile = /path/to/more/config
```

**command_address**

The email address that clist will use to accept commands.

**log**

Path to the file where clist will store log information.

**database**

The name of the sqlite database where clist will store data.

**smtp_hostname**

Hostname of the SMTP server that clist will use to send outgoing mail.

**smtp_port**

SMTP port number. Usually 25, 587, or 2525 for unencrypted connections, and 465 or 25025 for encrypted transport.

**smtp_username**  
**smtp_password**

Username and password for the SMTP server.

**Debug**

Not used.

**ConfigFile**

Path to a file containing configuration to be loaded by clist.

### List Configuration

Each list section looks like this:

```
List
name = unused?
archive = archive@example.com
owner = owner@example.com
description = A short description of the list.
Id = thelist
address = not super sure what this does
hidden = true|false
subscribers_only = true|false
posters,omitempty = authorized@example.com
bcc,omitempty = anthonyb@example.com
```

**name**

This appears to be unused.

**archive**

An email address. The value for this setting is set as the List-Archive header on outgoing messages. Per RFC 5064, this value described how to access archives for the list.

**owner**

An email address inserted into outgoing mail as the List-Owner header.

**description**

A description of this list. 
This value is provided in the listing of lists given in response to a request for a list of mailing lists.

**Id**

A short identifier for this list.
Used when subscribing or unsubscribing to the list.

**address**

I'm not entirely sure what this does.

**hidden**

If set to true, this list will not appear in the listing of mailing lists.

**subscribers_only**

If set to true, only subscribers may send messages to the list.

**posters,omitempty**

A comma-separated whitelist of approved posters.

**bcc,omitempty**

A comma-separated list of bcc recipients.

Using clist
-----------

clist monitors the subject line of emails sent to the command address.
This address is set using the `command_address` configuration setting.
To get a list of available commands, send an email with 'help' in the subject line to the address that clist uses to handle commands.

Available commands are:

**help**

Reply with a list of valid commands.

**lists**

Reply with a list of mailing lists. Any lists having `hidden = true` will be omitted from this list.

**subscribe**

The subscribe command must be followed by a valid list Id.
clist will add the address in the From head to the list following a confirmation.

**unsubscribe**

The unsubscribe command must be followed by a valid list Id.
clist will remove the address in the From head to the list.

## Contributing

Send patches and bug reports to j3s.

