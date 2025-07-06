package mqtt

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"testing"
	"time"

	"fmt"

	paho "github.com/eclipse/paho.mqtt.golang"
)

// helper to generate self-signed cert
func generateCert(t *testing.T) (certFile, keyFile, caFile string) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	tmpl := x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test"}, NotBefore: time.Now(), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	dir := t.TempDir()
	certFile = dir + "/cert.pem"
	keyFile = dir + "/key.pem"
	caFile = dir + "/ca.pem"
	if err := os.WriteFile(certFile, certPEM, 0644); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0644); err != nil {
		t.Fatalf("write key: %v", err)
	}
	if err := os.WriteFile(caFile, certPEM, 0644); err != nil {
		t.Fatalf("write ca: %v", err)
	}
	return
}

func TestLoadTLSConfig(t *testing.T) {
	cert, key, ca := generateCert(t)
	cfg := Config{UseTLS: true, ClientCert: cert, ClientKey: key, CABundle: ca}
	tlsCfg, err := cfg.LoadTLSConfig()
	if err != nil {
		t.Fatalf("load tls: %v", err)
	}
	if len(tlsCfg.Certificates) == 0 {
		t.Fatalf("no certs loaded")
	}
	if tlsCfg.RootCAs == nil {
		t.Fatalf("no root CAs")
	}
}

func TestNewClientOptionsAuth(t *testing.T) {
	opts, err := NewClientOptions(Config{Broker: "tcp://localhost:1883", ClientID: "id", Username: "u", Password: "p"})
	if err != nil {
		t.Fatalf("opts: %v", err)
	}
	if opts.Username != "u" || opts.Password != "p" {
		t.Fatalf("auth not set")
	}
}

func TestQoSSettings(t *testing.T) {
	mc := &mockClient{}
	newMQTTClient = func(o *paho.ClientOptions) pahoClient { mc.opts = o; return mc }
	defer func() { newMQTTClient = func(opts *paho.ClientOptions) pahoClient { return paho.NewClient(opts) } }()
	cfg := Config{Broker: "tcp://localhost:1883", ClientID: "id", AckTopic: "a", QoS: map[string]byte{"command": 2, "ack": 1}}
	cli, err := NewPahoClient(cfg)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	if len(mc.subscribed) == 0 || mc.subscribed[0].qos != 1 {
		t.Fatalf("subscribe qos not applied")
	}
	cmdID, err := cli.SendOrder("veh1", 10)
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if len(mc.published) == 0 || mc.published[0].qos != 2 {
		t.Fatalf("publish qos not applied")
	}
	// trigger ack
	payload := fmt.Sprintf(`{"command_id":"%s"}`, cmdID)
	cli.onAck(nil, mockMessage{[]byte(payload)})
	ok, err := cli.WaitForAck(cmdID, time.Millisecond)
	if err != nil || !ok {
		t.Fatalf("ack wait failed: %v", err)
	}
}

func TestLWTConfigured(t *testing.T) {
	mc := &mockClient{}
	newMQTTClient = func(o *paho.ClientOptions) pahoClient { mc.opts = o; return mc }
	defer func() { newMQTTClient = func(opts *paho.ClientOptions) pahoClient { return paho.NewClient(opts) } }()
	cfg := Config{Broker: "tcp://localhost:1883", ClientID: "id", LWTTopic: "lwt", LWTPayload: "bye", LWTQoS: 1}
	cli, err := NewPahoClient(cfg)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	if !mc.opts.WillEnabled {
		t.Fatalf("will not enabled")
	}
	if mc.opts.WillTopic != "lwt" || string(mc.opts.WillPayload) != "bye" {
		t.Fatalf("will options incorrect")
	}
	cli.Disconnect()
	if len(mc.published) != 0 {
		t.Fatalf("unexpected publish on disconnect")
	}
}

func TestRetryLogic(t *testing.T) {
	mc := &mockClient{publishErrs: []error{fmt.Errorf("net fail"), nil}}
	newMQTTClient = func(o *paho.ClientOptions) pahoClient { mc.opts = o; return mc }
	defer func() { newMQTTClient = func(opts *paho.ClientOptions) pahoClient { return paho.NewClient(opts) } }()
	cfg := Config{Broker: "tcp://localhost:1883", ClientID: "id", MaxRetries: 1, BackoffMS: 1}
	cli, err := NewPahoClient(cfg)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	_, err = cli.SendOrder("veh1", 1)
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if len(mc.published) != 2 {
		t.Fatalf("expected retries")
	}
}

func TestWaitForAckTimeout(t *testing.T) {
	mc := &mockClient{}
	newMQTTClient = func(o *paho.ClientOptions) pahoClient { mc.opts = o; return mc }
	defer func() { newMQTTClient = func(opts *paho.ClientOptions) pahoClient { return paho.NewClient(opts) } }()
	cfg := Config{Broker: "tcp://localhost:1883", ClientID: "id"}
	cli, err := NewPahoClient(cfg)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	cmdID, _ := cli.SendOrder("veh1", 1)
	ok, err := cli.WaitForAck(cmdID, time.Millisecond)
	if err == nil || ok {
		t.Fatalf("expected timeout")
	}
}

// mockClient implements pahoClient for tests
type mockClient struct {
	opts       *paho.ClientOptions
	subscribed []struct {
		topic string
		qos   byte
	}
	published []struct {
		topic string
		qos   byte
	}
	publishErrs []error
}

func (m *mockClient) IsConnected() bool { return true }
func (m *mockClient) Connect() paho.Token {
	if m.opts != nil && m.opts.OnConnect != nil {
		m.opts.OnConnect(m)
	}
	return &dummyToken{}
}
func (m *mockClient) Disconnect(uint) {}
func (m *mockClient) Publish(topic string, qos byte, _ bool, _ interface{}) paho.Token {
	m.published = append(m.published, struct {
		topic string
		qos   byte
	}{topic, qos})
	if len(m.publishErrs) > 0 {
		err := m.publishErrs[0]
		m.publishErrs = m.publishErrs[1:]
		return &dummyToken{err: err}
	}
	return &dummyToken{}
}
func (m *mockClient) Subscribe(topic string, qos byte, _ paho.MessageHandler) paho.Token {
	m.subscribed = append(m.subscribed, struct {
		topic string
		qos   byte
	}{topic, qos})
	return &dummyToken{}
}
func (m *mockClient) SubscribeMultiple(map[string]byte, paho.MessageHandler) paho.Token {
	return &dummyToken{}
}
func (m *mockClient) Unsubscribe(...string) paho.Token        { return &dummyToken{} }
func (m *mockClient) AddRoute(string, paho.MessageHandler)    {}
func (m *mockClient) OptionsReader() paho.ClientOptionsReader { return paho.ClientOptionsReader{} }
func (m *mockClient) IsConnectionOpen() bool                  { return true }

type dummyToken struct{ err error }

func (d dummyToken) Wait() bool                     { return true }
func (d dummyToken) WaitTimeout(time.Duration) bool { return true }
func (d dummyToken) Done() <-chan struct{}          { ch := make(chan struct{}); close(ch); return ch }
func (d dummyToken) Error() error                   { return d.err }

type mockMessage struct{ p []byte }

func (m mockMessage) Duplicate() bool   { return false }
func (m mockMessage) Qos() byte         { return 0 }
func (m mockMessage) Retained() bool    { return false }
func (m mockMessage) Topic() string     { return "" }
func (m mockMessage) MessageID() uint16 { return 0 }
func (m mockMessage) Payload() []byte   { return m.p }
func (m mockMessage) Ack()              {}
