package portforward

import (
	"errors"
	"io"
	"io/ioutil"
	"net"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/httpstream"

	"github.com/rusenask/k8s-portforward/pkg/recover"
)

// streamConn wraps a pair of SPDY streams and pretends to be a net.Conn
type streamConn struct {
	log         *logr.Logger
	c           httpstream.Connection
	dataStream  httpstream.Stream
	errorStream httpstream.Stream
	errch       chan error
}

var _ net.Conn = (*streamConn)(nil)

func newStreamConn(log logr.Logger, c httpstream.Connection, dataStream, errorStream httpstream.Stream) *streamConn {
	s := &streamConn{
		log:         &log,
		errch:       make(chan error, 1),
		c:           c,
		dataStream:  dataStream,
		errorStream: errorStream,
	}

	go s.readErrorStream()

	return s
}

func (s *streamConn) readErrorStream() {
	defer recover.Panic(s.log)

	message, err := ioutil.ReadAll(s.errorStream)
	if err != nil {
		s.errch <- err
	} else if len(message) > 0 {
		s.errch <- errors.New(string(message))
	} else {
		s.errch <- nil
	}
	close(s.errch)
}

func (s *streamConn) Read(b []byte) (int, error) {
	n, err := s.dataStream.Read(b)
	if err == io.EOF {
		err = <-s.errch
		if err == nil {
			err = io.EOF
		}
	}

	return n, err
}

func (s *streamConn) Write(b []byte) (int, error) {
	return s.dataStream.Write(b)
}

func (s *streamConn) Close() error {
	return s.c.Close()
}

func (s *streamConn) LocalAddr() net.Addr              { return nil }
func (s *streamConn) RemoteAddr() net.Addr             { return nil }
func (s *streamConn) SetDeadline(time.Time) error      { return errors.New("not implemented") }
func (s *streamConn) SetReadDeadline(time.Time) error  { return errors.New("not implemented") }
func (s *streamConn) SetWriteDeadline(time.Time) error { return errors.New("not implemented") }
