package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/blck-snwmn/gocacher"
	"github.com/blck-snwmn/gocacher/example/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/sync/singleflight"
)

type AuthorRepository struct {
	queries *db.Queries
	sfList  *singleflight.Group
	cache   *gocacher.Cache
}

func (r *AuthorRepository) Lists(ctx context.Context) ([]db.Author, error) {
	v, err := r.cache.Do(ctx, "lists", func(key string) (interface{}, time.Time, error) {
		v, err, _ := r.sfList.Do(key, func() (interface{}, error) {
			fmt.Println("called")
			return r.queries.ListAuthors(ctx)
		})
		return v, time.Now().Add(3 * time.Second), err
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
		cache:   gocacher.New(),
	}, nil
}

func main() {
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, "postgres://postgres:postgres@localhost:5432/postgres")
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, db.CreateAuthor)
	if err != nil {
		log.Fatalf("failed to create table: %v", err)
	}

	repo, err := newAuthorRepository(conn)
	if err != nil {
		log.Fatalf("failed to create repository: %v", err)
	}

	for i := 0; i < 10; i++ {
		_, err := repo.Create(ctx, fmt.Sprintf("name-%d", i), fmt.Sprintf("bio-%d", i))
		if err != nil {
			log.Fatalf("failed to create author: %v", err)
		}
	}

	var sg sync.WaitGroup
	for i := 0; i < 10; i++ {
		sg.Add(1)
		go func(i int) {
			log.Println("start")
			defer sg.Done()

			authors, err := repo.Lists(ctx)
			if err != nil {
				log.Fatalf("failed to list authors: %v", err)
			}
			if i != 0 {
				return
			}
			for _, author := range authors {
				fmt.Println(author)
			}
		}(i)
	}
	sg.Wait()

	log.Println("===done===")

	_, err = repo.Create(ctx, "name-99", "bio-99")
	if err != nil {
		log.Fatalf("failed to create author: %v", err)
	}
	authors, err := repo.Lists(ctx)
	if err != nil {
		log.Fatalf("failed to list authors: %v", err)
	}
	for _, author := range authors {
		fmt.Println(author)
	}

	log.Println("===done===")

	time.Sleep(5 * time.Second)

	authors, err = repo.Lists(ctx)
	if err != nil {
		log.Fatalf("failed to list authors: %v", err)
	}
	for _, author := range authors {
		fmt.Println(author)
	}

	// delete table
	_, err = conn.Exec(ctx, "DROP TABLE authors")
	if err != nil {
		log.Fatalf("failed to drop table: %v", err)
	}
}
