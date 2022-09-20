package wg

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	rawgql "github.com/Khan/genqlient/graphql"
	"github.com/fly-apps/terraform-provider-fly/graphql"
	"github.com/miekg/dns"
	"golang.org/x/crypto/curve25519"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/tun/netstack"
	"math/rand"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"time"
)

type PrivateKey device.NoisePrivateKey
type PublicKey device.NoisePublicKey

type Config struct {
	LocalPrivateKey PrivateKey
	LocalNetwork    *net.IPNet

	RemotePublicKey PublicKey
	RemoteNetwork   *net.IPNet

	Endpoint  string
	DNS       net.IP
	KeepAlive int
	MTU       int
	LogLevel  int
}

// ToHex I am not 100 percent sure if this is right, but I stole it from wireguard-go
func (pk PrivateKey) ToHex() string {
	const hex = "0123456789abcdef"
	buf := new(bytes.Buffer)
	buf.Reset()
	for i := 0; i < len(pk); i++ {
		buf.WriteByte(hex[pk[i]>>4])
		buf.WriteByte(hex[pk[i]&0xf])
	}
	return buf.String()
}

func (pk PublicKey) ToHex() string {
	const hex = "0123456789abcdef"
	buf := new(bytes.Buffer)
	buf.Reset()
	for i := 0; i < len(pk); i++ {
		buf.WriteByte(hex[pk[i]>>4])
		buf.WriteByte(hex[pk[i]&0xf])
	}
	return buf.String()
}

//FIXME(dov): we should not be panicing anywhere in here

func (pk *PrivateKey) UnmarshalText(text []byte) error {
	buf, err := base64.StdEncoding.DecodeString(string(text))
	if err != nil {
		return err
	}
	if len(buf) != device.NoisePrivateKeySize {
		return errors.New("invalid noise private key")
	}

	copy(pk[:], buf)
	return nil
}

func (pk *PublicKey) UnmarshalText(text []byte) error {
	buf, err := base64.StdEncoding.DecodeString(string(text))
	if err != nil {
		return err
	}
	if len(buf) != device.NoisePublicKeySize {
		return errors.New("invalid noise private key")
	}

	copy(pk[:], buf)
	return nil
}

type WireGuardState struct {
	Org          string
	Name         string
	Region       string
	LocalPublic  string
	LocalPrivate string
	DNS          string
	Token        string
	Peer         graphql.AddWireguardPeerAddWireGuardPeerAddWireGuardPeerPayload
}

func (s *WireGuardState) TunnelConfig() *Config {
	skey := PrivateKey{}
	if err := skey.UnmarshalText([]byte(s.LocalPrivate)); err != nil {
		panic(fmt.Sprintf("martian local private key: %s", err))
	}

	pkey := PublicKey{}
	if err := pkey.UnmarshalText([]byte(s.Peer.Pubkey)); err != nil {
		panic(fmt.Sprintf("martian local public key: %s", err))
	}

	//fmt.Println(fmt.Sprintf("%s/120", s.Peer.Peerip))
	_, lnet, err := net.ParseCIDR(fmt.Sprintf("%s/120", s.Peer.Peerip))
	if err != nil {
		panic(fmt.Sprintf("martian local public: %s/120: %s", s.Peer.Peerip, err))
	}

	raddr := net.ParseIP(s.Peer.Peerip).To16()
	for i := 6; i < 16; i++ {
		raddr[i] = 0
	}

	_, rnet, _ := net.ParseCIDR(fmt.Sprintf("%s/48", raddr))

	raddr[15] = 3
	dns := net.ParseIP(raddr.String())

	wgl := *lnet
	wgr := *rnet

	return &Config{
		LocalPrivateKey: skey,
		LocalNetwork:    &wgl,
		RemotePublicKey: pkey,
		RemoteNetwork:   &wgr,
		Endpoint:        s.Peer.Endpointip + ":51820",
		DNS:             dns,
	}
}

type Tunnel struct {
	dev       *device.Device
	tun       tun.Device
	net       *netstack.Net
	dnsIP     net.IP
	State     *WireGuardState
	Config    *Config
	apiClient *rawgql.Client

	wscancel func()
	resolv   *net.Resolver
}

func doConnect(ctx context.Context, state *WireGuardState, apiClient *rawgql.Client) (*Tunnel, error) {
	cfg := state.TunnelConfig()

	localNetworkIp, _ := netip.AddrFromSlice(cfg.LocalNetwork.IP)
	localIPs := []netip.Addr{localNetworkIp}
	dnsIP, _ := netip.AddrFromSlice(cfg.DNS)

	mtu := cfg.MTU
	if mtu == 0 {
		mtu = device.DefaultMTU
	}

	tunDev, gNet, err := netstack.CreateNetTUN(localIPs, []netip.Addr{dnsIP}, mtu)
	if err != nil {
		return nil, err
	}

	endpointHost, endpointPort, err := net.SplitHostPort(cfg.Endpoint)
	if err != nil {
		return nil, err
	}

	endpointIPs, err := net.LookupIP(endpointHost)
	if err != nil {
		return nil, err
	}

	endpointIP := endpointIPs[rand.Intn(len(endpointIPs))]
	endpointAddr := net.JoinHostPort(endpointIP.String(), endpointPort)

	wgDev := device.NewDevice(tunDev, conn.NewDefaultBind(), device.NewLogger(cfg.LogLevel, "(fly-ssh) "))

	wgConf := bytes.NewBuffer(nil)
	fmt.Println(cfg.RemoteNetwork)
	fmt.Println("ENDPOINT ADDR:", endpointAddr)
	fmt.Fprintf(wgConf, "private_key=%s\n", cfg.LocalPrivateKey.ToHex())
	fmt.Fprintf(wgConf, "public_key=%s\n", cfg.RemotePublicKey.ToHex())
	fmt.Fprintf(wgConf, "endpoint=%s\n", endpointAddr)
	fmt.Fprintf(wgConf, "allowed_ip=%s\n", cfg.RemoteNetwork)
	fmt.Fprintf(wgConf, "persistent_keepalive_interval=%d\n", cfg.KeepAlive)

	if err := wgDev.IpcSetOperation(bufio.NewReader(wgConf)); err != nil {
		return nil, err
	}
	err = wgDev.Up()
	if err != nil {
		return nil, err
	}

	return &Tunnel{
		dev:       wgDev,
		tun:       tunDev,
		net:       gNet,
		dnsIP:     cfg.DNS,
		Config:    cfg,
		State:     state,
		apiClient: apiClient,

		resolv: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				return gNet.DialContext(ctx, "tcp", net.JoinHostPort(dnsIP.String(), "53"))
			},
		},
	}, nil
}

func (t *Tunnel) Resolver() *net.Resolver {
	return t.resolv
}

func C25519pair() (string, string) {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)

	var private [32]byte
	_, err := r.Read(private[:])
	if err != nil {
		panic(fmt.Sprintf("reading from random: %s", err))
	}

	public, err := curve25519.X25519(private[:], curve25519.Basepoint)
	if err != nil {
		panic(fmt.Sprintf("can't mult: %s", err))
	}

	return base64.StdEncoding.EncodeToString(public),
		base64.StdEncoding.EncodeToString(private[:])
}

type Client struct {
	HttpClient http.Client
}

type Transport struct {
	underlyingTransport http.RoundTripper
	token               string
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	fmt.Println(req)
	req.Header.Add("Authorization", "Bearer "+t.token)
	return t.underlyingTransport.RoundTrip(req)
}

func (t *Tunnel) NewHttpClient() Client {
	net.DefaultResolver = t.resolv
	underlyingTransport := &http.Transport{
		DialContext: t.net.DialContext,
	}
	transport := Transport{
		token:               t.State.Token,
		underlyingTransport: underlyingTransport,
	}

	return Client{HttpClient: http.Client{Transport: &transport}}
}

func (t *Tunnel) NetStack() *netstack.Net {
	return t.net
}

func (t *Tunnel) Down() error {
	_, err := graphql.RemoveWireguardPeer(context.Background(), *t.apiClient, graphql.RemoveWireGuardPeerInput{
		OrganizationId: t.State.Org,
		Name:           t.State.Name,
	})
	if err != nil {
		// Silently ignore this error for now. It's not the end of the world if the peer isn't removed
	}
	if t.dev != nil {
		t.dev.Close()
	}
	t.dev, t.tun, t.net = nil, nil, nil
	return nil
}

func (t *Tunnel) LookupAAAA(ctx context.Context, name string) ([]net.IP, error) {
	var m dns.Msg
	_ = m.SetQuestion(dns.Fqdn(name), dns.TypeAAAA)

	r, err := t.QueryDNS(ctx, &m)
	if err != nil {
		return nil, err
	}

	results := make([]net.IP, 0, len(r.Answer))

	for _, a := range r.Answer {
		ip := a.(*dns.AAAA).AAAA
		results = append(results, ip)
	}

	return results, nil
}

func (t *Tunnel) QueryDNS(ctx context.Context, msg *dns.Msg) (*dns.Msg, error) {
	client := dns.Client{
		Net: "tcp",
		Dialer: &net.Dialer{
			Resolver: t.resolv,
		},
	}

	c, err := t.net.DialContext(ctx, "tcp", net.JoinHostPort(t.dnsIP.String(), "53"))
	if err != nil {
		return nil, err
	}
	defer c.Close()

	conn := &dns.Conn{Conn: c}
	defer conn.Close()

	r, _, err := client.ExchangeWithConn(msg, conn)
	return r, err
}

func Establish(ctx context.Context, org string, region string, token string, client *rawgql.Client) (*Tunnel, error) {
	peerName := "terraform-tunnel-" + strconv.FormatInt(time.Now().Unix(), 10)
	public, private := C25519pair()

	peer, err := graphql.AddWireguardPeer(ctx, *client, graphql.AddWireGuardPeerInput{
		OrganizationId: org,
		Region:         region,
		Name:           peerName,
		Pubkey:         public,
	})

	if err != nil {
		return nil, err
	}

	state := WireGuardState{
		Org:          org,
		Name:         peerName,
		Region:       region,
		LocalPrivate: private,
		LocalPublic:  public,
		Peer:         peer.AddWireGuardPeer,
		Token:        token,
	}
	tunnel, err := doConnect(ctx, &state, client)

	if err != nil {
		return nil, err
	}
	return tunnel, nil
}
