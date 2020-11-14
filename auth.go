package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
)

type key int

const (
	props key = iota
)

// Create the JWT key used to create the signature
var jwtKey = []byte("g78rw9etuiwletru9w87g")

// Credentials is the struct for holding email/pass/id combination
type Credentials struct {
	ID    int    `json:"uid"`
	Pass  string `json:"pass"`
	Email string `json:"email"`
}

// Claims is a part of token
type Claims struct {
	ID     int      `json:"uid"`
	Email  string   `json:"email"`
	jwt.StandardClaims
}

// AddAuthHandlers for main function handler
func AddAuthHandlers(r *mux.Router) {
	r.HandleFunc("/login/", LoginHandler).Methods("POST")
}

// LoginHandler Create the Signin handler
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var credsRequest, credsDB Credentials
	var user User
	// Get the JSON body and decode into credentials
	err := json.NewDecoder(r.Body).Decode(&credsRequest)
	if err != nil {
		// If the structure of the body is wrong, return an HTTP error
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	//only one tenant per uzer
	err = db.QueryRow(`SELECT users.uid, users.email, users.uid as users_uid, users.email as users_email
						FROM admin
						WHERE email=? AND pass=?`, credsRequest.Email, credsRequest.Pass).Scan(&credsDB.ID, &credsDB.Email, &user.Uid, &user.Email)

	if err != nil || credsDB.ID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Declare the expiration time of the token
	// here, we have kept it as 5 minutes
	expirationTime := time.Now().Add(100 * time.Hour)
	// Create the JWT claims, which includes the username and expiry time
	var userList []User
	userList = append(userList, user)

	claims := &Claims{
		Email:  credsRequest.Email,
		ID:     credsDB.ID,
		StandardClaims: jwt.StandardClaims{
			// In JWT, the expiry time is expressed as unix milliseconds
			ExpiresAt: expirationTime.Unix(),
		},
	}

	// Declare the token with the algorithm used for signing, and the claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Create the JWT string
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		// If there is an error in creating the JWT return an internal server error
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Finally, we set the client cookie for "token" as the JWT we just generated
	// we also set an expiry time which is the same as the token itself
	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   tokenString,
		Expires: expirationTime,
	})

	b, err := json.Marshal(struct {
		Token string `json:"token"`
	}{
		tokenString,
	})
	fmt.Fprintf(w, string(b))

}

// Middleware is a wrapper of handlers that needs to perform authentication
func Middleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := strings.Split(r.Header.Get("Authorization"), "Bearer ")
		if len(authHeader) != 2 {
			fmt.Println("Malformed token")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Malformed Token"))
		} else {
			jwtToken := authHeader[1]

			claims := &Claims{}
			tkn, err := jwt.ParseWithClaims(jwtToken, claims, func(token *jwt.Token) (interface{}, error) {
				return jwtKey, nil
			})
			if err != nil {
				if err == jwt.ErrSignatureInvalid {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if !tkn.Valid {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), props, claims)
			// Access context values in handlers like this
			// claims, _ := r.Context().Value("props").(*Claims)
			// fmt.Println(claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	})
}