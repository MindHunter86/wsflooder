package main

import (
	"log"
	"net/http"
	"time"
//	"fmt"
	"sync"
	"net"
//	"syscall"
	"runtime"
	"io/ioutil"
	"crypto/tls"
	"net/url"
	"os"
	"errors"
	"os/signal"
	"github.com/gorilla/websocket"
	"strconv"
)
var dstHost, dstOrigin string
func main() {
	l := log.New(os.Stdout, "[GENERIC]: ", log.Ldate | log.Ltime | log.Lmicroseconds)
	defer l.Print("Stopped!")
	l.Print("Started!")
	c := make( chan bool, 1 )
	u := &url.URL {
		Scheme: "ws",
		Host: "csgopuzo.com:7703",
        Path: "/socket.io/",
		RawQuery: "EIO=3&transport=websocket",
	}
	dstOrigin = "http://csgopuzo.com"
	dstHost = "csgopuzo.com:7703"
//	dstHost = "csgopuzo.com:7703"	// 7703 port is a chat
//	dstHost = "csgopuzo.com:7701"	// 7701 port is a roulette

/*
 *	cURL zone:

curl 'http://csgopuzo.com:7703/socket.io/?EIO=3&transport=polling' -H 'Host: csgopuzo.com:7703' -H 'User-Agent: Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:51.0) Gecko/20100101 Firefox/51.0' -H 'Accept-Language: en-US,en;q=0.5' --compressed -H 'Referer: http://csgopuzo.com/' -H 'Origin: http://csgopuzo.com' -H 'Cookie: lastupdate=1485879520' -H 'DNT: 1' -H 'Connection: keep-alive' -H 'Pragma: no-cache' -H 'Cache-Control: no-cache'

*/


	itr := make(chan os.Signal)
	signal.Notify( itr, os.Interrupt )
	w := &sync.WaitGroup{}
//	max_cpu := runtime.NumCPU()
	max_cpu := uint8(1)
	max_workers := uint8(1)
	runtime.GOMAXPROCS(int(max_cpu))

//	rl := &syscall.Rlimit{}
//	if e := syscall.Getrlimit(syscall.RLIMIT_NOFILE, rl); e == nil {
//		l.Print( "Finded system limits: ", rl )
//		rl.Cur = 10240
//		rl.Max = 10240
//		if e = syscall.Setrlimit(syscall.RLIMIT_NOFILE, rl); e == nil {
//			if e := syscall.Getrlimit(syscall.RLIMIT_NOFILE, rl); e == nil {
//				l.Print( "Rlimit updated: ", rl )
//			} else { l.Printf( "Error getting #2 rlimit! (%s)", e ); return }
//		} //else { l.Printf( "Error setting rlimit! (%s)", e ); return }
//	} else { l.Printf( "Error in getting rlimit! (%s)", e ); return }
//
//	var ips []net.Addr
//	if i, e := net.InterfaceByName("eth0"); e == nil {
//		if ips, e = i.Addrs(); e == nil {
//			if len( ips ) >= max_cpu+2 {
//				l.Print( "I will use this addresses: ", ips[2:10] )
//			} else { l.Print("I wil use systems ONE IP") }
//		} else {
//			l.Printf( "I can't get addresses from interface! (%s)", e )
//			return
//		}
//	} else {
//		l.Printf( "I can't get interface by name! (%s)", e )
//		return
//	}

	//	Worker spawning
	for i:=uint8(0); i<max_cpu; i++ {
		go func() {
//			var a *net.TCPAddr
//			if ip, _, e := net.ParseCIDR( ips[2+i].String() ); e == nil {
//				if a, e = net.ResolveTCPAddr( "tcp4", ip.String() + ":0" ); e != nil {
//					l.Printf( "Promblem with resolving TCPAddr for goroutine. Using system IP (%s)", e )
//				}
//			} else { l.Printf( "Problem with parsing CIDR! (%s)", e ) }

//			for k:=uint16(0); k<max_workers; k++ {
//			//	go worker( c, u, i, k, w, a )
//				go worker( c, u, i, k, w, nil )
//				time.Sleep( time.Millisecond * 500 )
//			}
			for k:=uint8(0); k<max_workers; k++ {
				go func() {
					z := &Worker{
						ch: c,
						url: u,
						cpu: i,
						thd: k,
						wg: w,
						ip: nil,
					}
					if e := z.Spawn(); e != nil { l.Fatalf( "FERR: WORKER %d-%d exited with non-nil code! | %s", i, k, e.Error() ) }
				}()
			}
		}()
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


type Worker struct {
	ch chan bool
	url *url.URL
	cpu, thd uint8
	wg *sync.WaitGroup
	ip *net.TCPAddr
}





func (w *Worker) Spawn() (error) {
	w.wg.Add(1)
	log.Println( "DEBUG: " + strconv.Itoa(int(w.cpu)) )
	l := log.New( os.Stdout, "Worker #" + strconv.Itoa(int(w.cpu)) + "-" + strconv.Itoa(int(w.thd)) + " ", log.Ldate | log.Ltime | log.Lmicroseconds )
	l.Println("INF: Worker has been STARTED")
	defer w.wg.Done()
	defer l.Println("INF: Worker has been STOPPED")

// Old "connector"
	hd := &http.Header{}
	hd.Set( "Host", dstHost )
	hd.Set( "Origin", dstOrigin )
	hd.Set( "Referer", dstOrigin )
	hd.Set( "User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:48.0) Gecko/20100101 Firefox/48.0" )
	// &tls.Config{InsecureSkipVerify: true}

	var wd *websocket.Dialer
	if w.ip == nil {
		wd = &websocket.Dialer{
			Proxy: http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	} else {
		nd := &net.Dialer{ LocalAddr: w.ip }
		wd = &websocket.Dialer{ NetDial: func( n, a string ) ( net.Conn, error ) { return nd.Dial( "tcp", a ) } }
	}

	l.Println("INF: websocket dialer has been inited! Connecting to dst ...")
	cn, _, e := wd.Dial( w.url.String(), *hd )
	if e != nil {
		return errors.New( "ERR: Connection failed! | " + e.Error() )
	} else {
		l.Println("INF: Connected to DST!")
	}

	tck := time.NewTicker( time.Second * 1 )
	defer tck.Stop()

	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		defer wg.Done()
		defer cn.Close()
		defer l.Println("INF: Reader end!")
		l.Println("INF: Reader has been spawned!")

		for {
			_, rdr, e := cn.NextReader(); if e != nil {
				l.Println( "ERR: Colud not read message from socket! | ", e.Error() ) ; return
			}
			mg, e := ioutil.ReadAll(rdr); if e != nil {
				l.Println( "ERR: could not read message from io reader! | " + e.Error() ) ; return
			}
			if len(mg) == 0 { l.Println("WRN: Received empty message!"); continue }

			switch(mg[0]) {
			case '3':
				l.Println("INF: Received PONG from DST!")
				if e := cn.WriteMessage( websocket.TextMessage, []byte("5") ); e != nil {
					l.Println( "ERR: could not send UPGRADE after PONG to DST! | " + e.Error() ); return
				}
			case 4:
				l.Println( "INF: Recevied message from server! | ", string(mg) )
			default:
				l.Println( "WRN: Unknown message type from DST! | ", string(mg) )
			}
		}
	}()

	for {
		select {
		case c:=<-w.ch:
			if c == false {
				l.Println("Catched DROP signal. Stopping ...")
				if e := cn.Close(); e != nil { return errors.New( "ERR: Could not close socket! | " + e.Error() ) }
				l.Println("INF: Waiting workers's goroutine ...")
				wg.Wait()
				return nil
			}
		case <-tck.C:
			if e = cn.WriteMessage( websocket.TextMessage, []byte("2") ); e != nil {
				defer cn.Close()
				return errors.New( "ERR: could not write message in socket! | " + e.Error() )
			}
		}
	}

	l.Println("INF: Disconnected from DST!")
	return cn.Close()
}


//func worker(c chan bool, u *url.URL, nc int, nt uint16, w *sync.WaitGroup, a *net.TCPAddr ) {
//	defer w.Done()
//	w.Add(1)
//	l := log.New(os.Stdout, fmt.Sprintf( "[Worker #%d-%d] ", nc, nt ), log.Ldate | log.Ltime | log.Lmicroseconds)
//	defer l.Print("INF: Worker stopped!")
//	l.Print( "INF: Worker has been initialized!" )
//	for {
//		select {
//		case cm:=<-c:
//			switch(cm) {
//			case false:
//				l.Print("Worker has been blocked by DROP signal")
//				return
//			}
//		default:
//			l.Print(" I'm spawning connector ...")
//			if ! connector(c, u, l, a) {
//				l.Printf( "I have promblems with my connector. Respawning it ..." )
//				time.Sleep(time.Second * 5)
//			} else {
//				l.Print("I recieve DROP signal!")
//				return
//			}
//		}
//	}
//}
//
//func connector(ch chan bool, u *url.URL, l *log.Logger, a *net.TCPAddr) ( bool ) {
//	for {
//		select {
//		case <-wsping.C:
//			if e := c.WriteMessage( websocket.TextMessage, []byte("2") ); e != nil {
//				defer c.Close()
//				l.Printf( "Something wrong in pinger! (%s)", e )
//				return false
//			}
//		case cm := <-ch:
//			l.Println("test message from connector by nonDROP signal")
//			switch(cm) {
//			case false:
//				defer c.Close()
//				defer wsping.Stop()
//				l.Print("Connector received DROP signal from worker!")
//				return true
//			default:
//				l.Println("test message from connector by nonDROP signal")
//			}
//		default: // reader
//			l.Println("SPAMER if non block")
//			var m []byte
//			if _, r, e := c.NextReader() ; e == nil {
//				if m, e =	ioutil.ReadAll(r); e == nil {
//					if len(m) == 0 { l.Print("Readed empty message"); continue }
//				} else {
//					defer c.Close()
//					l.Println( "Error in reading message from IO reader: ", e )
//					return false
//				}
//			} else {
//				defer c.Close()
//				l.Println( "Error in reading message: ", e )
//				return false
//			}
//		// if all ifs are OK then we have valid message
//			switch(m[0]) {
//			case '3':
//				l.Print("Received PONG from server")
//				if e = c.WriteMessage( websocket.TextMessage, []byte("5") ); e != nil {
//					l.Println( "Some errors after upgrade sending: ", e ); return false
//				} else { l.Println( "IO UPGRADE has been sended!" ) }
//			case '4':
//				l.Printf( "Meassage from server: %s", m )
//			default:
//				l.Printf( "Undefined message from server! (%s)", m )
//				if e = c.WriteMessage( websocket.TextMessage, []byte("2") ); e != nil {
//					l.Println( "Some errors after ping sending: ", e ); return false
//				} else { l.Println( "IO PING has been sended!" ) }
//			}
//
//
//
////		//	Old shit:
////			l.Println("SPAMER if non block")
////			if _, m, e := c.ReadMessage(); e == nil {
////				if len(m) == 0 {
////					defer c.Close()
////					l.Print("Message has zero length");
////					return false
////				} else { l.Print(string(m)) }
//////				} else {
//////					for z:=0 ; z<10; z++ {
//////						c.WriteMessage( websocket.TextMessage, m )
//////					}
//////				}
////
////				switch( m[0] ) {
////				case '3':
////					l.Print("Received PONG from server")
////					if e = c.WriteMessage( websocket.TextMessage, []byte("5") ); e != nil {
////						l.Println( "Some errors after upgrade sending: ", e ); return false
////					} else { l.Println( "IO UPGRADE has been sended!" ) }
////				case '4':
////					l.Printf( "Meassage from server: %s", m )
////				default:
////					l.Printf( "Undefined message from server! (%s)", m )
////					if e = c.WriteMessage( websocket.TextMessage, []byte("2") ); e != nil {
////						l.Println( "Some errors after ping sending: ", e ); return false
////					} else { l.Println( "IO PING has been sended!" ) }
////				}
////			} else {
////				defer c.Close()
////				l.Printf( "Something wrong in reader! (%s)", e )
////				return false
////			}
//		}
//	}
//}
