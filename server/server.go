package main

import (
	// Note: Also remove the 'os' import.

	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/sessions"
	"github.com/srinathgs/mysqlstore"
)

const keyServerAddr = "serverAddr"

var store *mysqlstore.MySQLStore

func handleRoot(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	fmt.Printf("%s: got / request\n", ctx.Value(keyServerAddr))
	io.WriteString(w, "<h1>This is the ant game!!!!!!!!</h1>")
	io.WriteString(w, "<p>Get your grass <a href='/trade'>here</a></p>")
}
func buyGrass(amt int, sess *sessions.Session) string {
	sess.Values["totalGrass"] = amt + sess.Values["totalGrass"].(int)
	return "Bought: " + fmt.Sprint(amt)

}
func sellGrass(amt int, sess *sessions.Session) string {
	totalGrass := sess.Values["totalGrass"].(int)
	if totalGrass >= amt {
		sess.Values["totalGrass"] = totalGrass - amt
		return "Sold: " + fmt.Sprint(amt)
	} else {
		return "Failed to sell"
	}

}
func handleTrade(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	session, _ := store.Get(r, "antsTrading")
	if session.Values["totalGrass"] == nil {
		session.Values["totalGrass"] = 0
	}

	fmt.Printf("%s: got /trade request\n", ctx.Value(keyServerAddr))
	var message string
	if r.URL.Query().Get("action") == "buy_grass" {
		q, err := strconv.Atoi(r.FormValue("quantityOfGrass"))
		if err == nil {
			message = buyGrass(q, session)
		}
	} else if r.URL.Query().Get("action") == "sell_grass" {
		q, err := strconv.Atoi(r.FormValue("quantityOfGrass"))
		if err == nil {
			message = sellGrass(q, session)
		}
	}
	err := session.Save(r, w)
	fmt.Printf("%#v\n", session)
	if err != nil {
		fmt.Println("session.save error = ", err)
	}
	io.WriteString(w, "<h1>hello Bozo </h1>"+
		"<p>You are 69 years old</p>"+
		"<p>Are you trading today?</p>")
	io.WriteString(w, "<form method='POST'>"+
		"  <label>Buy or Sell Grass Quantity</label>"+
		"  <input type='text' name='quantityOfGrass' />"+
		"  <button formaction='?action=buy_grass'>Buy</button>"+
		"  <button formaction='?action=sell_grass'>Sell</button>"+
		"</form>")
	io.WriteString(w, "<h4>"+message+"</h4>")

	io.WriteString(w, "<h4>Total grass: "+fmt.Sprint(session.Values["totalGrass"])+"</h4>")

	io.WriteString(w, "<pre>Get:")
	io.WriteString(w, fmt.Sprint(r.URL.Query()))
	io.WriteString(w, "</pre>")

	io.WriteString(w, "<pre>Post:")
	io.WriteString(w, fmt.Sprint(r.Form))
	io.WriteString(w, "</pre>")

}

func main() {
	var err error
	store, err = mysqlstore.NewMySQLStore("ants:REDACTED@tcp(127.0.0.1:3306)/ants?parseTime=true&loc=Local", "sess", "/", 3600, []byte("MySecret"))
	if err != nil {
		panic(err)
	}
	defer store.Close()

	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/trade", handleTrade)

	err = http.ListenAndServe(":8080", nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}
