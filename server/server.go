package main

import (
	// Note: Also remove the 'os' import.

	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
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

func getResourceQuantity(user_id int, resource_id int) int {
	var quantity int = 0
	err := db.QueryRowContext(dbctx, "SELECT quantity FROM inventory_item WHERE user_id = ? and resource_id = ?", user_id, resource_id).Scan(&quantity)
	if err == sql.ErrNoRows {
		_, err2 := db.ExecContext(dbctx, "insert into inventory_item(user_id,resource_id,quantity) values(?,?,?)", user_id, resource_id, quantity)
		if err2 != nil {
			panic(err)
		}
	} else if err != nil {
		panic(err)
	}
	return quantity
}

const (
	GRASS = 1
)

func buyGrass(user_id int, amt int) (string, int) {
	_, err := db.ExecContext(dbctx, "insert into inventory_item(user_id,resource_id,quantity) values(?,1,?)"+
		" on duplicate key update quantity = quantity + ?", user_id, amt, amt)
	if err != nil {
		panic(err)
	}
	quantity := getResourceQuantity(user_id, GRASS)
	return "Bought: " + fmt.Sprint(amt), quantity
}
func sellGrass(user_id int, amt int) (string, int) {
	quantity := getResourceQuantity(user_id, GRASS)
	if amt > quantity {
		return "Failed to sell", quantity
	}
	_, err := db.ExecContext(dbctx, "update inventory_item set quantity = quantity - ? where user_id = ? and resource_id = 1", amt, user_id)
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
	if !validEmail(email) {
		return fmt.Errorf("email invalid")
	}
	if err := validatePassword(password); err != nil {
		return err
	}
	insertResult, err := db.ExecContext(dbctx, "INSERT into user(name,email,password) values(?,?,?)", name, email, password_hash)
	if err != nil {
		return fmt.Errorf("failed to create account")
	}
	id, _ = insertResult.LastInsertId()
	sess.Values["loggedInUserId"] = int(id)
	fmt.Printf("Valid login id = %d\n", id)
	return nil
}
func validEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password too short (min 8 characters)")
	}
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
		panic(err)
	}
	if session.Values["loggedInUserId"] != nil {
		http.Redirect(w, r, "/seb/ants/trade", http.StatusFound)
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
		http.Redirect(w, r, "/seb/ants/trade", http.StatusFound)
		return
	}
	component := templates.Register(errmsg)
	component.Render(context.Background(), w)
}

func buySellOperator(r *http.Request, user_id int, bsGrass func(int, int) (string, int)) (string, int) {
	var message string
	var quantity int
	q, err := strconv.Atoi(r.FormValue("quantityOfGrass"))
	if err == nil {
		message, quantity = bsGrass(user_id, q)
	} else {
		message = "Invalid quantity"
		quantity = getResourceQuantity(user_id, GRASS)
	}
	return message, quantity
}

func handleTrade(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "antsTrading")
	if session.Values["loggedInUserId"] == nil {
		http.Redirect(w, r, "/seb/ants/login", http.StatusFound)
		return
	}
	user_id := session.Values["loggedInUserId"].(int)
	var err error
	var username string
	err = db.QueryRowContext(dbctx, "SELECT name FROM user WHERE id = ?", user_id).Scan(&username)
	if err != nil {
		session.Values["loggedInUserId"] = nil
		http.Redirect(w, r, "/seb/ants/login", http.StatusFound)
		err = session.Save(r, w)
		if err != nil {
			fmt.Println("session.save error = ", err)
		}
		return
	}
	var message string
	var quantity int
	if r.URL.Query().Get("action") == "buy_grass" {
		message, quantity = buySellOperator(r, user_id, buyGrass)
	} else if r.URL.Query().Get("action") == "sell_grass" {
		message, quantity = buySellOperator(r, user_id, sellGrass)
	} else {
		quantity = getResourceQuantity(user_id, GRASS)
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

	http.HandleFunc("/seb/ants/", handleRoot)
	http.HandleFunc("/seb/ants/trade", handleTrade)
	http.HandleFunc("/seb/ants/login", handleLogin)
	http.HandleFunc("/seb/ants/register", handleRegister)

	err = http.ListenAndServe("127.0.0.1:8080", nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}
