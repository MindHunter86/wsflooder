package main

import (
	"log"
	"time"
	"strings"
	"net/http"
	"fmt"
	"bytes"
	"encoding/base64"
	"crypto/md5"

	"github.com/gorilla/mux"
)
func iferror(err error) { if err != nil { log.Fatal("ERROR: ", err) } }

func main() {
	log.Print( "Prg started!" )
	rt := mux.NewRouter()
	rt.HandleFunc("/", HelloWorld)
	log.Fatal(http.ListenAndServe(":8081", rt))
}

// HelloWorld root handler
func HelloWorld(w http.ResponseWriter, r *http.Request) {
	res, tm := secreted()
	w.Write([]byte("Helly Melly!\n"))
	w.Write([]byte("https://pma.mh00.info/secure/50x.html?secl="+res+"&sect="+tm+"\n"))
}

func secreted() (string, string) {
	var buf bytes.Buffer

	timed := fmt.Sprint( time.Now().Unix() + 30 )
  timer := "1473025133"
  log.Println(timer)

	buf.WriteString("secret")
	buf.WriteString("/secure/")
	buf.WriteString("50x.html")
	buf.WriteString( timed )
	buf.WriteString("95.28.208.177")
	log.Print("BUF: ", buf.String())

	md5hash := md5.Sum( buf.Bytes() )
	log.Printf("MD5: %x", md5hash[:])

	base64hash := base64.StdEncoding.EncodeToString( md5hash[:] )
	log.Print("BASE64: ", base64hash)

	stringed := strings.Replace( strings.Replace(base64hash, "+", "-", -1), "/", "_", -1)
	log.Print("STRINGED: ", stringed)

	result := strings.Replace(stringed, "=", "", -1)
	log.Print( "RESULT: ", result )
	log.Print("https://pma.mh00.info/secure/50x.html?secl=", result, "&sect=", timed)
	return result, timed
}

// // $secure_link_expires$uri$remote_addr secret
// echo -n '1472411763/secure/info.html127.0.0.1 secret' | openssl md5 -binary | openssl base64 | tr +/ -_ | tr -d =
/// QAm2KPVxx_SROSt_9WapTQ

// // mysecret$uri$arg_sect$remote_addr
// echo -n 'secret/secret/info.html1472411763127.0.0.1' | openssl md5 -binary | openssl base64 | tr +/ -_ | tr -d =
/// e-bm7r2e_QXyCZX2TRmwkg


/*
$name = "mult.flv";
$secret = 'mysecretword';
$time = time() + 10800; //ссылка будет рабочей три часа
$key = str_replace("=", "", strtr(base64_encode(md5($secret.'/realvideo/'.$name.$time.getenv("REMOTE_ADDR"), TRUE)), "+/", "-_"));
$encoded_url = "http://site.ru/video/$key/$time/$name";
*/
