package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/blck-snwmn/gocacher/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/sync/singleflight"
)

type AuthorRepository struct {
	queries *db.Queries
	sfList  *singleflight.Group
}

func (r *AuthorRepository) Lists(ctx context.Context) ([]db.Author, error) {
	v, err, _ := r.sfList.Do("lists", func() (interface{}, error) {
		fmt.Println("called")
		return r.queries.ListAuthors(ctx)
	})
	if err != nil {
		return nil, err
	}
	return v.([]db.Author), nil
}

func (r *AuthorRepository) Create(ctx context.Context, name string, bio string) (db.Author, error) {
	return r.queries.CreateAuthor(ctx, db.CreateAuthorParams{
		Name: name,
		Bio:  pgtype.Text{String: bio, Valid: true},
	})
}

func newAuthorRepository(conn *pgx.Conn) (*AuthorRepository, error) {
	return &AuthorRepository{
		queries: db.New(conn),
		sfList:  &singleflight.Group{},
	}, nil
}

func main() {
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, "postgres://postgres:postgres@localhost:5432/postgres")
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer conn.Close(ctx)

	// _, err = conn.Exec(ctx, db.CreateAuthor)
	// if err != nil {
	// 	log.Fatalf("failed to create table: %v", err)
	// }

	repo, err := newAuthorRepository(conn)
	if err != nil {
		log.Fatalf("failed to create repository: %v", err)
	}

	// for i := 0; i < 10; i++ {
	// 	_, err := repo.Create(ctx, fmt.Sprintf("name-%d", i), fmt.Sprintf("bio-%d", i))
	// 	if err != nil {
	// 		log.Fatalf("failed to create author: %v", err)
	// 	}
	// }

	var sg sync.WaitGroup
	for i := 0; i < 10; i++ {
		sg.Add(1)
		go func() {
			log.Println("start")
			defer sg.Done()

			_, err := repo.Lists(ctx)
			if err != nil {
				log.Fatalf("failed to list authors: %v", err)
			}
		}()
	}
	sg.Wait()
	log.Println("done")

	authors, err := repo.Lists(ctx)
	if err != nil {
		log.Fatalf("failed to list authors: %v", err)
	}
	for _, author := range authors {
		fmt.Println(author)
	}
}
