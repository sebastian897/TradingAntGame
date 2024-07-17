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
)

const keyServerAddr = "serverAddr"

var store = sessions.NewFilesystemStore("sess")

func handleRoot(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	fmt.Printf("%s: got / request\n", ctx.Value(keyServerAddr))
	io.WriteString(w, "<h1>This is the ant game!!!!!!!!</h1>")
	io.WriteString(w, "<p>Get your grass <a href='/trade'>here</a></p>")
}
func buyGrass(w http.ResponseWriter, amt int) {
	io.WriteString(w, "<h4>Bought: "+fmt.Sprint(amt)+"</h4>")
}
func sellGrass(w http.ResponseWriter, amt int) {
	io.WriteString(w, "<h4>Sold: "+fmt.Sprint(amt)+"</h4>")
}
func handleTrade(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	fmt.Printf("%s: got /hello request\n", ctx.Value(keyServerAddr))
	io.WriteString(w, "<h1>hello Bozo </h1>"+
		"<p>You are 69 years old</p>"+
		"<p>Are you trading today?</p>")
	io.WriteString(w, "<form method='POST'>"+
		"  <label>Buy or Sell Grass Quantity</label>"+
		"  <input type='text' name='quantityOfGrass' />"+
		"  <button formaction='?action=buy_grass'>Buy</button>"+
		"  <button formaction='?action=sell_grass'>Sell</button>"+
		"</form>")

	if r.URL.Query().Get("action") == "buy_grass" {
		q, err := strconv.Atoi(r.FormValue("quantityOfGrass"))
		if err == nil {
			buyGrass(w, q)
		}
	} else if r.URL.Query().Get("action") == "sell_grass" {
		q, err := strconv.Atoi(r.FormValue("quantityOfGrass"))
		if err == nil {
			sellGrass(w, q)
		}
	}

	io.WriteString(w, "<pre>Get:")
	io.WriteString(w, fmt.Sprint(r.URL.Query()))
	io.WriteString(w, "</pre>")

	io.WriteString(w, "<pre>Post:")
	io.WriteString(w, fmt.Sprint(r.Form))
	io.WriteString(w, "</pre>")
}
func main() {
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/trade", handleTrade)

	err := http.ListenAndServe(":8080", nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}
