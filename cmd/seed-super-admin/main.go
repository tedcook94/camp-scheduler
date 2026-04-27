package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"camp-scheduler/internal/admin"
	"camp-scheduler/internal/config"

	"golang.org/x/term"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	if cfg.Auth.ServerURL == "" || cfg.Auth.SharedSecret == "" {
		fmt.Fprintln(os.Stderr, "error: AUTH_SERVER_URL and AUTH_SHARED_SECRET must be set")
		os.Exit(1)
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Create a super-admin user")
	fmt.Println("-------------------------")

	username := prompt(reader, "Username")
	email := prompt(reader, "Email")
	firstName := prompt(reader, "First name")
	lastName := prompt(reader, "Last name")
	password := promptPassword("Password")

	syncer := admin.NewOrgSyncer(admin.OrgSyncerConfig{
		BaseURL: cfg.Auth.InternalURL(),
		Secret:  cfg.Auth.SharedSecret,
		Timeout: 10 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID, err := syncer.CreateUser(ctx, admin.CreateUserRequest{
		Email:     email,
		Password:  password,
		Username:  username,
		FirstName: firstName,
		LastName:  lastName,
		Role:      "super_admin",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating super-admin: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nSuper-admin created: %s (%s) [%s]\n", username, email, userID)
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
		first := readPasswordOnce(label)
		if first == "" {
			fmt.Println("  Password cannot be empty.")
			continue
		}
		confirm := readPasswordOnce("Confirm " + strings.ToLower(label))
		if first != confirm {
			fmt.Println("  Passwords do not match.")
			continue
		}
		return first
	}
}

func readPasswordOnce(label string) string {
	fmt.Printf("%s: ", label)
	raw, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading password: %v\n", err)
		os.Exit(1)
	}
	return strings.TrimSpace(string(raw))
}
