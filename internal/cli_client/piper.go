package cli_client

import (
	"github.com/cbeuw/Cloak/internal/common"
	"github.com/cbeuw/Cloak/libcloak/client"
	"io"
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

func RouteUDP(bindFunc func() (*net.UDPConn, error), streamTimeout time.Duration, singleplex bool, newSeshFunc func() *client.CloakClient) {
	var cloakClient *client.CloakClient
	localConn, err := bindFunc()
	if err != nil {
		log.Fatal(err)
	}

	streams := make(map[string]net.Conn)
	var streamsMutex sync.Mutex

	data := make([]byte, 8192)
	for {
		i, addr, err := localConn.ReadFrom(data)
		if err != nil {
			log.Errorf("Failed to read first packet from proxy client: %v", err)
			continue
		}

		if !singleplex && (cloakClient == nil || cloakClient.IsClosed()) {
			cloakClient = newSeshFunc()
		}

		streamsMutex.Lock()
		stream, ok := streams[addr.String()]
		if !ok {
			if singleplex {
				cloakClient = newSeshFunc()
			}

			stream, err = cloakClient.Dial()
			if err != nil {
				if singleplex {
					cloakClient.Close()
				}
				log.Errorf("Failed to open stream: %v", err)
				streamsMutex.Unlock()
				continue
			}
			streams[addr.String()] = stream
			streamsMutex.Unlock()

			_ = stream.SetReadDeadline(time.Now().Add(streamTimeout))

			proxyAddr := addr
			go func(stream net.Conn, localConn *net.UDPConn) {
				buf := make([]byte, 8192)
				for {
					n, err := stream.Read(buf)
					if err != nil {
						log.Tracef("copying stream to proxy client: %v", err)
						break
					}
					_ = stream.SetReadDeadline(time.Now().Add(streamTimeout))

					_, err = localConn.WriteTo(buf[:n], proxyAddr)
					if err != nil {
						log.Tracef("copying stream to proxy client: %v", err)
						break
					}
				}
				streamsMutex.Lock()
				delete(streams, addr.String())
				streamsMutex.Unlock()
				stream.Close()
				return
			}(stream, localConn)
		} else {
			streamsMutex.Unlock()
		}

		_, err = stream.Write(data[:i])
		if err != nil {
			log.Tracef("copying proxy client to stream: %v", err)
			streamsMutex.Lock()
			delete(streams, addr.String())
			streamsMutex.Unlock()
			stream.Close()
			continue
		}
		_ = stream.SetReadDeadline(time.Now().Add(streamTimeout))
	}
}

func RouteTCP(listener net.Listener, streamTimeout time.Duration, singleplex bool, newSeshFunc func() *client.CloakClient) {
	var cloakClient *client.CloakClient
	for {
		localConn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
			continue
		}
		if !singleplex && (cloakClient == nil || cloakClient.IsClosed()) {
			cloakClient = newSeshFunc()
		}
		go func(sesh *client.CloakClient, localConn net.Conn, timeout time.Duration) {
			if singleplex {
				sesh = newSeshFunc()
			}

			data := make([]byte, 10240)
			_ = localConn.SetReadDeadline(time.Now().Add(streamTimeout))
			i, err := io.ReadAtLeast(localConn, data, 1)
			if err != nil {
				log.Errorf("Failed to read first packet from proxy client: %v", err)
				localConn.Close()
				return
			}
			var zeroTime time.Time
			_ = localConn.SetReadDeadline(zeroTime)

			stream, err := sesh.Dial()
			if err != nil {
				log.Errorf("Failed to open stream: %v", err)
				localConn.Close()
				if singleplex {
					sesh.Close()
				}
				return
			}

			_, err = stream.Write(data[:i])
			if err != nil {
				log.Errorf("Failed to write to stream: %v", err)
				localConn.Close()
				stream.Close()
				return
			}

			go func() {
				if _, err := common.Copy(localConn, stream); err != nil {
					log.Tracef("copying stream to proxy client: %v", err)
				}
			}()
			if _, err = common.Copy(stream, localConn); err != nil {
				log.Tracef("copying proxy client to stream: %v", err)
			}
		}(cloakClient, localConn, streamTimeout)
	}
}