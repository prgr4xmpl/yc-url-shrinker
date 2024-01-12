package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"path"
	"regexp"

	"github.com/gorilla/mux"
	"github.com/ydb-platform/ydb-go-sdk/v3"
	"github.com/ydb-platform/ydb-go-sdk/v3/table"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/options"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/types"
)

var (
	short = regexp.MustCompile(`[a-zA-Z0-9]{8}`)
	long  = regexp.MustCompile(`https?://(?:[-\w.]|%[\da-fA-F]{2})+`)
)

func hashString(s string) (string, error) {
	hasher := fnv.New32a()
	_, err := hasher.Write([]byte(s))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func isShortCorrect(link string) bool {
	return short.FindStringIndex(link) != nil
}

func isLongCorrect(link string) bool {
	return long.FindStringIndex(link) != nil
}

type service struct {
	db     *ydb.Driver
	router *mux.Router
}

func getService(ctx context.Context, dsn string, opts ...ydb.Option) (s *service, err error) {
	s = &service{
		router: mux.NewRouter(),
	}

	s.db, err = ydb.Open(ctx, dsn, opts...)
	if err != nil {
		err = fmt.Errorf("Couldn't connect to db: %v", err)
		return
	}

	if err = s.createTable(ctx); err != nil {
		s.db.Close(ctx)
		err = fmt.Errorf("Couldn't create table: %v", err)
		return
	}

	s.router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, world!"))
	}).Methods(http.MethodGet)
	s.router.HandleFunc("/shorten", s.handleShoren).Methods(http.MethodPost)

	return s, nil
}

func (s *service) Close(ctx context.Context) {
	s.db.Close(ctx)
	return
}

func (s *service) createTable(ctx context.Context) (err error) {
	err = s.db.Table().Do(ctx, func(ctx context.Context, session table.Session) error {
		return session.CreateTable(ctx, path.Join(s.db.Name(), "urls"),
			options.WithColumn("src", types.TypeUTF8),
			options.WithColumn("hash", types.TypeUTF8),
			options.WithPrimaryKeyColumn("hash"),
		)
	})
	return
}

func (s *service) insertShorten(ctx context.Context, url string) (hash string, err error) {
	hash, err = hashString(url)
	if err != nil {
		return
	}

	query := fmt.Sprintf(`
		REPLACE INTO
			urls (hash, src)
		VALUES
			('%s', '%s');
	`, url, hash)
	println(query)

	err = s.db.Table().Do(ctx, func(ctx context.Context, session table.Session) error {
		return session.ExecuteSchemeQuery(ctx, query)
	})

	return
}

func writeResponse(w http.ResponseWriter, statusCode int, body string) {
	w.WriteHeader(statusCode)
	w.Write([]byte(body))
}

func (s *service) handleShoren(w http.ResponseWriter, r *http.Request) {
	url, err := io.ReadAll(r.Body)
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if !isLongCorrect(string(url)) {
		err = fmt.Errorf("'%s' is not a valid URL", url)
		writeResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	hash, err := s.insertShorten(r.Context(), string(url))
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/text")
	writeResponse(w, http.StatusOK, hash)
}
