package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"camp-scheduler/internal/config"
	"camp-scheduler/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(context.Background(), cfg.Database.DSN())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error connecting to database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Create a super-admin user")
	fmt.Println("-------------------------")

	username := prompt(reader, "Username")
	email := prompt(reader, "Email")
	firstName := prompt(reader, "First name")
	lastName := prompt(reader, "Last name")
	password := promptPassword("Password")

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error hashing password: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	queries := db.New(pool)
	user, err := queries.CreateUser(ctx, db.CreateUserParams{
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		FirstName:    firstName,
		LastName:     lastName,
		Role:         "super_admin",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating user: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nSuper-admin created: %s (%s)\n", user.Username, user.Email)
}

func prompt(reader *bufio.Reader, label string) string {
	for {
		fmt.Printf("%s: ", label)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nerror reading input: %v\n", err)
			os.Exit(1)
		}
		input = strings.TrimSpace(input)
		if input != "" {
			return input
		}
		fmt.Println("  Value cannot be empty.")
	}
}

func promptPassword(label string) string {
	for {
		fmt.Printf("%s: ", label)
		raw, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading password: %v\n", err)
			os.Exit(1)
		}
		password := strings.TrimSpace(string(raw))
		if password != "" {
			return password
		}
		fmt.Println("  Password cannot be empty.")
	}
}
