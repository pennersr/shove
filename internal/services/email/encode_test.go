package email

import (
	"testing"
)

var png1Pixel = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x37, 0x6e, 0xf9,
	0x24, 0x00, 0x00, 0x00, 0x10, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x62, 0x60, 0x01, 0x00, 0x00,
	0x00, 0xff, 0xff, 0x03, 0x00, 0x00, 0x06, 0x00, 0x05, 0x57, 0xbf, 0xab, 0xd4, 0x00, 0x00, 0x00,
	0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}

func TestEncodeEmailDigest(t *testing.T) {
	emails := []email{
		{
			Subject: "Hello Jane!",
			From:    "john@doe.org",
			To:      []string{"jane@doe.org"},
			Text:    "Hello Jane!\n\nBye\n",
			HTML:    "<p>Hello <b>Jane!</b></p>",
			Attachments: []attachment{
				{
					Filename:    "px1.png",
					ContentType: "image/png",
					Content:     png1Pixel,
				},
			},
		},
		{
			Subject: "One last thing...!",
			From:    "john@doe.org",
			To:      []string{"jane@doe.org"},
			Text:    "Don't forget the milk...",
			HTML:    "<p>Don't forget the <b>milk</b></p>",
			Attachments: []attachment{
				{
					Filename:    "px1.png",
					ContentType: "image/png",
					Content:     png1Pixel,
				},
			},
		},
	}

	out, err := encodeEmailDigest(emails)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(out))
}

func TestEncodeEmail(t *testing.T) {
	e := email{

		Subject: "Hello!",
		From:    "john@doe.org",
		To:      []string{"jane@doe.org"},
		Text:    "Hello\n\nBye\n",
		HTML:    "<p>Hello <b>world</b></p>",
		Attachments: []attachment{
			{
				Filename:    "px1.png",
				ContentType: "image/png",
				Content:     png1Pixel,
			},
		},
	}
	out, err := encodeEmail(e)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(out))
}

func TestEncodeSMTPAddresses(t *testing.T) {
	from, to, err := encodeSMTPAddresses("John <john@doe.org>", []string{"jane@doe.org", "Nobody <noreply@mail.org>"})
	if err != nil {
		t.Fatal(err)
	}
	if from != "john@doe.org" {
		t.Fatal(from)
	}
	if to[0] != "jane@doe.org" {
		t.Fatal(to[0])
	}
	if to[1] != "noreply@mail.org" {
		t.Fatal(to[0])
	}
}

func TestEncodeSMTPAddressesErrors(t *testing.T) {
	_, _, err := encodeSMTPAddresses("John <john@doe.org", []string{"jane@doe.org", "Nobody <noreply@mail.org>"})
	if err == nil {
		t.Fail()
	}
	_, _, err = encodeSMTPAddresses("John <john@doe.org>", []string{"<jane@doe.org", "Nobody <noreply@mail.org>"})
	if err == nil {
		t.Fail()
	}
}
