package main

import (
	// Note: Also remove the 'os' import.

	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"
	"github.com/srinathgs/mysqlstore"
)

var store *mysqlstore.MySQLStore

func handleRoot(w http.ResponseWriter, r *http.Request) {
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

func Login(username string, password string, sess *sessions.Session) error {
	db, err := sql.Open("mysql", "ants:REDACTED@tcp(127.0.0.1:3306)/ants")
	if err != nil {
		panic(err)
	}
	// See "Important settings" section.
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	var id int
	err = db.QueryRowContext(ctx, "SELECT id FROM user WHERE name = ? and password = ?", username, password).Scan(&id)
	if err == sql.ErrNoRows {
		fmt.Printf("Invalid login %s\n", username)
		return fmt.Errorf("username or password invalid")
	} else if err != nil {
		panic(err)
	}
	sess.Values["loggedInUserId"] = id
	fmt.Printf("Valid login id = %d\n", id)
	return nil
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "antsTrading")
	var err error
	var errmsg string
	if r.URL.Query().Get("action") == "login" {
		u := r.FormValue("username")
		p := r.FormValue("password")
		err = Login(u, p, session)
		if err != nil {
			errmsg = err.Error()
		}
	}
	err = session.Save(r, w)
	if err != nil {
		fmt.Println("session.save error = ", err)
	}
	if session.Values["loggedInUserId"] != nil {
		http.Redirect(w, r, "/trade", http.StatusFound)
	}
	io.WriteString(w, "<h1>This is the ant game!!!!!!!!</h1>")
	io.WriteString(w, "<p>Login please.</p>")
	io.WriteString(w, "<form method='POST'>"+
		"  <label>Username</label>"+
		"  <input type='text' name='username'/>"+
		"  <label>Password</label>\n"+
		"  <input type='text' name='password' />"+
		"  <button formaction='?action=login'>Login</button>"+
		"</form>")
	io.WriteString(w, "<p style='color:red;'>"+errmsg+"</p>")
}
func handleTrade(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "antsTrading")
	if session.Values["loggedInUserId"] == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
	}
	if session.Values["totalGrass"] == nil {
		session.Values["totalGrass"] = 0
	}
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
	http.HandleFunc("/login", handleLogin)

	err = http.ListenAndServe(":8080", nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}
