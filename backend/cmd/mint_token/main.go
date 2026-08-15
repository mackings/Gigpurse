package main

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	secret := os.Getenv("JWT_SECRET")
	userID := os.Getenv("TEST_USER_ID")
	role := os.Getenv("TEST_ROLE")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID, "role": role, "exp": time.Now().Add(1 * time.Hour).Unix(),
	})
	s, err := token.SignedString([]byte(secret))
	if err != nil {
		panic(err)
	}
	fmt.Println(s)
}
