package main

import (
	"context"
	"go-htmx/components"
	"go-htmx/database"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/utils"
	"golang.org/x/crypto/bcrypt"
)

var queries *database.Queries

func main() {
	// Create a context to handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Create a WaitGroup to keep track of running goroutines
	var wg sync.WaitGroup

	// Start the HTTP server
	wg.Add(1)
	go start_server(ctx, &wg)

	// Listen for termination signals
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	<-signalCh

	// Cancel the context to signal the HTTP server to stop
	cancel()

	// Wait for the HTTP server to finish
	wg.Wait()

}

func start_server(ctx context.Context, wg *sync.WaitGroup) {
	// Initialize database
	defer wg.Done()
	db, err := database.NewDBConnection()
	if err != nil {
		log.Fatal(err)
	}
	dbConnection := db.GetDB()
	queries = database.New(dbConnection)

	_, err = dbConnection.Exec(`
				CREATE TABLE IF NOT EXISTS users (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						username TEXT UNIQUE NOT NULL,
						password TEXT NOT NULL
				)`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = dbConnection.Exec(`
				CREATE TABLE IF NOT EXISTS sessions (
						session_id TEXT PRIMARY KEY,
						user_id INTEGER NOT NULL,
						expires_at INTEGER NOT NULL,
						FOREIGN KEY (user_id) REFERENCES users(id)
				)`)
	if err != nil {
		log.Fatal(err)
	}

	store := session.New()

	// Create new Fiber app
	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("queries", queries)
		return c.Next()
	})

	app.Use(logger.New())

	app.Get("/", func(c *fiber.Ctx) error {
		sess, err := store.Get(c)
		if err != nil {
			// Empty error handler - this should be handled
			return err // Add this line
		}
		sessionID := sess.Get("session_id")
		if sessionID != nil {
			return render(c, components.Home(sessionID.(string)))
		}
		return render(c, components.Home(""))
	})

	app.Get("/login", func(c *fiber.Ctx) error {
		return render(c, components.Login())
	})

	app.Post("/login", func(c *fiber.Ctx) error {
		type LoginInput struct {
			Username string `form:"username"`
			Password string `form:"password"`
		}

		start := time.Now()
		defer func() {
			elapsed := time.Since(start)
			if elapsed < 3*time.Second {
				time.Sleep(3*time.Second - elapsed)
			}
		}()

		var input LoginInput
		if err := c.BodyParser(&input); err != nil {
			return err
		}

		user, err := queries.GetUserByUsername(c.Context(), input.Username)
		if err != nil {
			return render(c, components.Login(withError("Invalid username or password")))
		}

		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password))
		if err != nil {
			return render(c, components.Login(withError("Invalid username or password")))
		}

		sess, err := store.Get(c)
		if err != nil {
			return err
		}
		sessionID := utils.UUID()
		sess.Set("session_id", sessionID)
		if err := sess.Save(); err != nil {
			return err
		}

		expiresAt := time.Now().Add(5 * time.Minute).Unix()
		_, err = queries.CreateSession(c.Context(), database.CreateSessionParams{
			SessionID: sessionID,
			UserID:    user.ID,
			ExpiresAt: expiresAt,
		})
		if err != nil {
			return err
		}

		return c.Redirect("/")
	})

	app.Get("/register", func(c *fiber.Ctx) error {
		return render(c, components.Register())
	})

	app.Post("/register", func(c *fiber.Ctx) error {
		type RegisterInput struct {
			Username string `form:"username"`
			Password string `form:"password"`
		}

		var input RegisterInput
		if err := c.BodyParser(&input); err != nil {
			return err
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		queries := c.Locals("queries").(*database.Queries)
		_, err = queries.CreateUser(c.Context(), database.CreateUserParams{
			Username: input.Username,
			Password: string(hashedPassword),
		})

		if err != nil {
			return render(c, components.Register(withError("Username already exists: "+err.Error())))
		}

		return c.Redirect("/login")
	})

	app.Post("/logout", func(c *fiber.Ctx) error {
		sess, err := store.Get(c)
		if err != nil {
			return err
		}
		sessionID := sess.Get("session_id")
		if sessionID != nil {
			err = queries.DeleteSession(c.Context(), sessionID.(string))
			if err != nil {
				return err
			}
		}
		sess.Destroy()
		return c.Redirect("/")
	})

	// Serve static files
	app.Static("/dist", "./dist")

	// Start server
	go func() {
		log.Println("Server starting at http://localhost:8080")
		if err := app.Listen(":8080"); err != nil {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()

	// Shutdown the server gracefully
	log.Println("Server shutting down...")
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Fatal(err)
	}
	log.Println("Server shutdown complete.")

}

func render(c *fiber.Ctx, t templ.Component) error {
	c.Set("Content-Type", "text/html")
	return t.Render(c.Context(), c)
}

func withError(msg string) string {
	return msg
}
