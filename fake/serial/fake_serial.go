package serial

import (
	log "github.com/Sirupsen/logrus"
)

var logger = log.WithFields(log.Fields{
	"pkg": "utils",
})

type FakeSerial struct {
}

func (s FakeSerial) Read(p []byte) (n int, err error) {
	logger.Debugf("read %d bytes", len(p))
	return 0, nil
}

func (s FakeSerial) Write(p []byte) (n int, err error) {
	logger.Debugf("write: %v", p)
	return len(p), nil
}

func (s FakeSerial) Close() error {
	logger.Debugf("serial")
	return nil
}
