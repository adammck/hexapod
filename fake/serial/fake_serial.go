package serial

import (
	"bytes"
	log "github.com/Sirupsen/logrus"
)

var logger = log.WithFields(log.Fields{
	"pkg": "utils",
})

type FakeSerial struct {
	bytes.Buffer
}

func (s *FakeSerial) Read(p []byte) (n int, err error) {
	logger.Debugf("read %d bytes", len(p))
	return s.Buffer.Read(p)
}

func (s *FakeSerial) Write(p []byte) (n int, err error) {
	logger.Debugf("write: %v", p)

	// Respond to any PING
	if p[4] == 0x1 {
		s.Buffer.Write([]byte{
			0xff, // header
			0xff, // header
			p[2], // id
			2,    // params+2
			0,    // errbits
			0,    // checksum
		})
	}

	return len(p), nil
}

func (s *FakeSerial) Close() error {
	logger.Debugf("serial")
	return nil
}
