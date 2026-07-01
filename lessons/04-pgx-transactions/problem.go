//go:build ignore

// problem.go shows the NAIVE approach BEFORE the withTx refactor.
// Build tag "ignore" keeps it out of the package — it's for reading, not running.

package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// transferNaive is what most beginners write.
// Problems:
//  1. BEGIN/COMMIT/ROLLBACK boilerplate in every handler
//  2. If you add a third DB call later, easy to forget to use tx
//  3. The defer Rollback is missing — connection leaks on early return
func transferNaive(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// start transaction
		tx, err := pool.Begin(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "begin tx"})
			return
		}
		// BUG: no defer tx.Rollback — if anything below fails without explicit Rollback,
		// the connection is held open until the pool timeout.

		_, err = tx.Exec(c.Request.Context(),
			`UPDATE accounts SET balance = balance - $1 WHERE id = $2`, 100, 1)
		if err != nil {
			_ = tx.Rollback(c.Request.Context()) // remembered here...
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		_, err = tx.Exec(c.Request.Context(),
			`UPDATE accounts SET balance = balance + $1 WHERE id = $2`, 100, 2)
		if err != nil {
			_ = tx.Rollback(c.Request.Context()) // ...and here...
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// ...and what if someone adds a third call and forgets to Rollback on error?
		// With WithTx the defer handles all paths automatically.

		if err := tx.Commit(c.Request.Context()); err != nil {
			_ = tx.Rollback(c.Request.Context())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "commit"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}
