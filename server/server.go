package main

import (
	// Note: Also remove the 'os' import.

	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"ants/server/templates"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"
	"github.com/srinathgs/mysqlstore"
	"golang.org/x/crypto/bcrypt"
)

var store *mysqlstore.MySQLStore
var db *sql.DB
var dbctx = context.Background()

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// VerifyPassword verifies if the given password matches the stored hash.
func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func buyGrass(user_id int, amt int) (string, int) {
	var quantity int
	_, err := db.ExecContext(dbctx, "insert into inventory_item(user_id,resource_id,quantity) values(?,1,?)"+
		" on duplicate key update quantity = quantity + ?", user_id, amt, amt)
	if err != nil {
		panic(err)
	}
	err = db.QueryRowContext(dbctx, "SELECT quantity FROM inventory_item WHERE user_id = ? and resource_id = 1", user_id).Scan(&quantity)
	if err != nil {
		panic(err)
	}
	return "Bought: " + fmt.Sprint(amt), quantity
}
func sellGrass(user_id int, amt int) (string, int) {
	var quantity int
	err := db.QueryRowContext(dbctx, "SELECT quantity FROM inventory_item WHERE user_id = ? and resource_id = 1", user_id).Scan(&quantity)
	if err == sql.ErrNoRows {
		_, err = db.ExecContext(dbctx, "insert into inventory_item(user_id,resource_id,quantity) values(?,1,0)", user_id)
		if err != nil {
			panic(err)
		}
		return "Failed to sell", 0
	}
	if amt > quantity {
		return "Failed to sell", quantity
	}
	_, err = db.ExecContext(dbctx, "insert into inventory_item(user_id,resource_id,quantity) values(?,1,?)"+
		" on duplicate key update quantity = quantity - ?", user_id, amt, amt)
	if err != nil {
		panic(err)
	}
	return "Sold: " + fmt.Sprint(amt), quantity - amt
}
func Login(email string, password string, sess *sessions.Session) error {
	var password_hash string
	var id int
	err := db.QueryRowContext(dbctx, "SELECT id,password FROM user WHERE email = ?", email).Scan(&id, &password_hash)
	if err == sql.ErrNoRows || !VerifyPassword(password, password_hash) {
		fmt.Printf("Invalid login %s\n", email)
		return fmt.Errorf("email or password invalid")
	} else if err != nil {
		panic(err)
	}
	sess.Values["loggedInUserId"] = id
	fmt.Printf("Valid login id = %d\n", id)
	return nil
}

func Register(email string, password string, name string, sess *sessions.Session) error {
	var password_hash, _ = HashPassword(password)
	var id int64
	insertResult, err := db.ExecContext(dbctx, "INSERT into user(name,email,password) values(?,?,?)", name, email, password_hash)
	if err != nil {
		fmt.Printf("Invalid registeration %s\n", email)
		return fmt.Errorf("email invalid")
	}
	id, _ = insertResult.LastInsertId()
	sess.Values["loggedInUserId"] = int(id)
	fmt.Printf("Valid login id = %d\n", id)
	return nil
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	component := templates.Root()
	component.Render(context.Background(), w)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "antsTrading")
	var err error
	var errmsg string
	if r.URL.Query().Get("action") == "login" {
		u := r.FormValue("email")
		p := r.FormValue("password")
		err = Login(u, p, session)
		if err != nil {
			errmsg = err.Error()
		}
	} else if r.URL.Query().Get("action") == "logout" {
		session.Values["loggedInUserId"] = nil
	}
	err = session.Save(r, w)
	if err != nil {
		fmt.Println("session.save error = ", err)
	}
	if session.Values["loggedInUserId"] != nil {
		http.Redirect(w, r, "/trade", http.StatusFound)
		return
	}
	component := templates.Login(errmsg)
	component.Render(context.Background(), w)

}
func handleRegister(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "antsTrading")
	var err error
	var errmsg string
	if r.URL.Query().Get("action") == "register" {
		e := r.FormValue("email")
		n := r.FormValue("name")
		p := r.FormValue("password")
		err = Register(e, p, n, session)
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
		return
	}
	component := templates.Register(errmsg)
	component.Render(context.Background(), w)
}
func handleTrade(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "antsTrading")
	if session.Values["loggedInUserId"] == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	user_id := session.Values["loggedInUserId"].(int)
	var err error
	var username string
	err = db.QueryRowContext(dbctx, "SELECT name FROM user WHERE id = ?", user_id).Scan(&username)
	if err != nil {
		session.Values["loggedInUserId"] = nil
		http.Redirect(w, r, "/login", http.StatusFound)
		err = session.Save(r, w)
		if err != nil {
			fmt.Println("session.save error = ", err)
		}
		return
	}
	var message string
	var quantity int
	if r.URL.Query().Get("action") == "buy_grass" {
		q, err := strconv.Atoi(r.FormValue("quantityOfGrass"))
		if err == nil {
			message, quantity = buyGrass(user_id, q)
		}
	} else if r.URL.Query().Get("action") == "sell_grass" {
		q, err := strconv.Atoi(r.FormValue("quantityOfGrass"))
		if err == nil {
			message, quantity = sellGrass(user_id, q)
		}
	}
	err = session.Save(r, w)
	if err != nil {
		fmt.Println("session.save error = ", err)
	}
	component := templates.Trade(username, message, quantity)
	component.Render(context.Background(), w)
}

func main() {
	var err error
	store, err = mysqlstore.NewMySQLStore("ants:REDACTED@tcp(127.0.0.1:3306)/ants?parseTime=true&loc=Local", "sess", "/", 3600, []byte("MySecret"))
	if err != nil {
		panic(err)
	}
	defer store.Close()
	db, err = sql.Open("mysql", "ants:REDACTED@tcp(127.0.0.1:3306)/ants")
	if err != nil {
		panic(err)
	}
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/trade", handleTrade)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/register", handleRegister)

	err = http.ListenAndServe("localhost:8080", nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}
