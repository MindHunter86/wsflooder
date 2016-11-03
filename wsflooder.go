package main

import (
	"log"
	"net/http"
	"time"
	"fmt"
	"sync"
	"net"
	"syscall"
	"runtime"

	"net/url"
	"os"
	"os/signal"
	"github.com/gorilla/websocket"
)

func main() {
	l := log.New(os.Stdout, "[MOTHER THREAD]: ", log.Ldate | log.Ltime | log.Lmicroseconds)
	defer l.Print("Stopped!")
	l.Print("Started!")
	c := make( chan bool, 1 )
	u := &url.URL {
		Scheme: "ws",
		Host: "46.105.42.220:2082",
        Path: "/socket.io/",
		RawQuery: "EIO=3&transport=websocket",
	}
	itr := make(chan os.Signal)
	signal.Notify( itr, os.Interrupt )
	w := &sync.WaitGroup{}
//	max_cpu := runtime.NumCPU()
	max_cpu := 1
	runtime.GOMAXPROCS(max_cpu)

	rl := &syscall.Rlimit{}
	if e := syscall.Getrlimit(syscall.RLIMIT_NOFILE, rl); e == nil {
		l.Print( "Finded system limits: ", rl )
		rl.Cur = 10240
		rl.Max = 10240
		if e = syscall.Setrlimit(syscall.RLIMIT_NOFILE, rl); e == nil {
			if e := syscall.Getrlimit(syscall.RLIMIT_NOFILE, rl); e == nil {
				l.Print( "Rlimit updated: ", rl )
			} else { l.Printf( "Error getting #2 rlimit! (%s)", e ); return }
		} else { l.Printf( "Error setting rlimit! (%s)", e ); return }
	} else { l.Printf( "Error in getting rlimit! (%s)", e ); return }

	var ips []net.Addr
	if i, e := net.InterfaceByName("eno1"); e == nil {
		if ips, e = i.Addrs(); e == nil {
			if len( ips ) >= max_cpu+2 {
				l.Print( "I will use this addresses: ", ips[2:10] )
			} else { l.Print("I wil use systems ONE IP") }
		} else {
			l.Printf( "I can't get addresses from interface! (%s)", e )
			return
		}
	} else {
		l.Printf( "I can't get interface by name! (%s)", e )
		return
	}

	//	Worker spawning
	for i:=0; i<max_cpu; i++ {
		go func( i int ) {
			var a *net.TCPAddr
			if ip, _, e := net.ParseCIDR( ips[2+i].String() ); e == nil {
				if a, e = net.ResolveTCPAddr( "tcp4", ip.String() + ":0" ); e != nil {
					l.Printf( "Promblem with resolving TCPAddr for goroutine. Using system IP (%s)", e )
				}
			} else { l.Printf( "Problem with parsing CIDR! (%s)", e ) }

			for k:=uint16(0); k<1; k++ {
				go worker( c, u, i, k, w, a )
				time.Sleep( time.Millisecond * 500 )
			}
		}(i)
	}

	//go worker( c, u, 0, 0, w )

	for {
		select {
		case <-itr:
			l.Print("Received INTERRUPT from kernel!!!")
			close(c)
			l.Print("DROP signals have been sended to workers!")
			w.Wait()
			return
		}
	}
	os.Exit(0)
}


func worker(c chan bool, u *url.URL, nc int, nt uint16, w *sync.WaitGroup, a *net.TCPAddr ) {
	defer w.Done()
	w.Add(1)
	l := log.New(os.Stdout, fmt.Sprintf( "[Worker #%d-%d] ", nc, nt ), log.Ldate | log.Ltime | log.Lmicroseconds)
	defer l.Print("INF: Worker stopped!")
	l.Print( "INF: Worker has been initialized!" )
	for {
		select {
		case cm:=<-c:
			switch(cm) {
			case false:
				l.Print("Worker has been blocked by DROP signal")
				return
			}
		default:
			l.Print(" I'm spawning connector ...")
			if ! connector(c, u, l, a) {
				l.Printf( "I have promblems with my connector. Respawning it ..." )
				time.Sleep(time.Second * 5)
			} else {
				l.Print("I recieve DROP signal!")
				return
			}
		}
	}
}

func connector(ch chan bool, u *url.URL, l *log.Logger, a *net.TCPAddr) ( bool ) {
	h := make( http.Header )
	h.Set("Origin", u.Host)
	h.Set("Host", "csgf.ru:2082")
	h.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:48.0) Gecko/20100101 Firefox/48.0")

	var wd *websocket.Dialer
	if a == nil {
		wd = &websocket.Dialer {
			Proxy: http.ProxyFromEnvironment,
		}
	} else {
		nd := &net.Dialer { LocalAddr: a }
		wd = &websocket.Dialer {
			NetDial: func( n, a string ) ( net.Conn, error ) {
				if cn, e := nd.Dial("tcp", a); e == nil {
					return cn, nil
				} else { return nil, e }
			},
		}
	}

	l.Printf("Connector initialized! Connecting to %s ...", u.Host)

	c, _, e := wd.Dial(u.String(), h)
	if e == nil {
		l.Print("Connection established successfully!")
	} else { l.Printf("Connection failed! (%s)", e); return false }
	defer l.Printf( "Disconnected from %s successfully!", u.Host )

	if c == nil {
		l.Print( "I have very strange situation! Kill me, please!" )
		return false
	}
	defer c.Close() // For only established connection!

	wsping := time.NewTicker( time.Second * 10 )
	defer wsping.Stop()

	for {
		select {
		case <-wsping.C:
			if e := c.WriteMessage( websocket.TextMessage, []byte("2") ); e != nil {
				defer c.Close()
				l.Printf( "Something wrong in pinger! (%s)", e )
				return false
			}
		case cm := <-ch:
			switch(cm) {
			case false:
				defer c.Close()
				defer wsping.Stop()
				l.Print("Connector received DROP signal from worker!")
				return true
			}
		default: // reader
			if _, m, e := c.ReadMessage(); e == nil {
				if len(m) == 0 {
					defer c.Close()
					l.Print("Message has zero length");
					return false
				} else { }
//				} else { c.WriteMessage( websocket.TextMessage, m ) }

				switch( m[0] ) {
				case '3':
					l.Print("Received PONG from server")
				case '4':
					l.Printf( "Meassage from server: %s", m )
				default:
					l.Printf( "Undefined message from server! (%s)", m )
				}
			} else {
				defer c.Close()
				l.Printf( "Something wrong in reader! (%s)", e )
				return false
			}
		}
	}
}
