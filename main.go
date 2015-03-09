package main

import (
	"bytes"
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/smtp"
	"os"
	"strings"
)

func main() {
	var (
		user    = flag.String("user", "", "User name")
		pass    = flag.String("pass", "", "Password")
		host    = flag.String("host", "", "Host name")
		port    = flag.String("port", "", "Port")
		from    = flag.String("from", "", "From")
		recep   = flag.String("recep", "", "Comma delimited list of recipients")
		subject = flag.String("subj", "", "Subject")
		file    = flag.String("file", "", "File to attach")
		message = flag.String("msg", "", "Message to send. If message is '-' (dash) then read from stdin")
	)
	flag.Parse()
	if *host == "" {
		fmt.Print("Error: Must supply a host!\n\n")
		flag.Usage()
		os.Exit(1)
	}
	if *port == "" {
		fmt.Print("Error: Must supply a port!\n\n")
		flag.Usage()
		os.Exit(1)
	}
	if *recep == "" {
		fmt.Print("Error: Must supply a recep!\n\n")
		flag.Usage()
		os.Exit(1)
	}

	var body []byte
	var err error

	switch {
	case *message == "-":
		body, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Printf("Trouble reading from stdin:%v", err)
			os.Exit(1)
		}
	case *message == "":
		body = []byte(*message)
	default:
		body = []byte(*message)
	}

	msg := Message{
		To:      strings.Split(*recep, ","),
		From:    *from,
		Body:    body,
		Subject: *subject,
	}

	if *file != "" {
		err = msg.AddAttachment(*file)
		if err != nil {
			log.Printf("Trouble attaching file:%v", err)
			os.Exit(1)
		}
	}

	buf := new(bytes.Buffer)
	_, err = msg.WriteTo(buf)
	if err != nil {
		log.Printf("Trouble writing message to buffer:%v", err)
		os.Exit(1)
	}

	var a smtp.Auth
	if *user != "" {
		a = smtp.PlainAuth("", *user, *pass, *host)
	}

	log.Print("Sending mail...")
	//err = smtp.SendMail(fmt.Sprintf("%s:%s", *host, *port), smtp.PlainAuth("", *user, *pass, *host), msg.From, msg.To, buf.Bytes())
	err = SendMail(fmt.Sprintf("%s:%s", *host, *port), a, msg.From, msg.To, buf.Bytes())
	if err != nil {
		log.Printf("Trouble sending mail:%v", err)
		os.Exit(1)
	}
	log.Print("Sent.")
}

// copied from smtp package
func SendMail(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
	// connect to the remote SMTP server
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()

	if ok, _ := c.Extension("STARTTLS"); ok {
		host, _, _ := net.SplitHostPort(addr)
		config := &tls.Config{ServerName: host, InsecureSkipVerify: true}
		if err = c.StartTLS(config); err != nil {
			return err
		}
	}

	if a != nil {
		if err = c.Auth(a); err != nil {
			return err
		}
	}

	// Set the sender and recipient first
	if err := c.Mail(from); err != nil {
		return err
	}
	for _, addr := range to {
		if err := c.Rcpt(addr); err != nil {
			return err
		}
	}

	// Send the email body.
	wc, err := c.Data()
	if err != nil {
		return err
	}
	_, err = wc.Write(msg)
	if err != nil {
		return err
	}
	err = wc.Close()
	if err != nil {
		return err
	}

	// Send the QUIT command and close the connection.
	return c.Quit()
}
