package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"net/textproto"
	"path"
	"strings"
)

// Message is a simplified view of an email message
type Message struct {
	To          []string
	From        string
	Subject     string
	Body        []byte
	Attachments []Attachment
}

// Attachment represents a file attached to the email
type Attachment struct {
	Filename string
	Data     []byte
}

// Header is a helper type for handling email headers
type Header map[string][]string

// WriteTo writes the Header to the given io.Writer.
// It conforms to the io.WriterTo interface
func (h *Header) WriteTo(w io.Writer) (n int64, err error) {
	buf := new(bytes.Buffer)
	for k, v := range *h {
		val := strings.Join(v, "; ")
		fmt.Fprintf(buf, "%s: %s\r\n", k, val)
	}
	nn, err := w.Write(buf.Bytes())
	n = int64(nn)
	return n, err
}

// WriteTo writes the Message to the given io.Writer.
// It conforms to the io.WriterTo interface
func (m *Message) WriteTo(w io.Writer) (n int64, err error) {

	buf := new(bytes.Buffer)

	// start with the header
	header := Header{
		"To":      m.To,
		"From":    []string{m.From},
		"Subject": []string{m.Subject},
	}

	if m.Attachments != nil || len(m.Attachments) > 0 {
		// we have attachments: use a multipart mime email
		wr := multipart.NewWriter(buf)

		header["MIME-Version"] = []string{"1.0"}
		header["Content-Type"] = []string{"multipart/mixed", fmt.Sprintf("boundary=%s", wr.Boundary())}

		// write out the headers
		header.WriteTo(buf)

		// write the body part
		bodyheader := textproto.MIMEHeader{}
		bodyheader.Add("Content-Type", "text/plain")
		bhw, err := wr.CreatePart(bodyheader)
		if err != nil {
			return 0, err
		}

		bhw.Write(m.Body)

		// write out all attachments
		for a := range m.Attachments {
			att := m.Attachments[a]
			ext := path.Ext(att.Filename)
			mimetype := mime.TypeByExtension(ext)

			// file header
			mimeheader := textproto.MIMEHeader{}
			mimeheader.Add("Content-Type", mimetype)
			mimeheader.Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", att.Filename))

			mimehw, err := wr.CreatePart(mimeheader)
			if err != nil {
				log.Printf("trouble creating part:%v", err)
			}

			mimehw.Write(att.Data)

		}
		// close the multipart writer
		wr.Close()
	} else {
		// write a regular message
		header.WriteTo(buf)
		buf.WriteString("\r\n")
		buf.Write(m.Body)
	}

	nn, err := buf.WriteTo(w)
	return int64(nn), err
}

//
func (m *Message) AddAttachment(filepath string) error {
	fb, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}

	if m.Attachments == nil {
		m.Attachments = []Attachment{}
	}

	m.Attachments = append(m.Attachments, Attachment{
		path.Base(filepath),
		fb,
	})

	return nil
}
