package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type dbLogger struct {
	db     *sql.DB
	tracer trace.Tracer
}

func newDbLogger(db *sql.DB) DBTX {
	tracer := otel.Tracer("db")
	return &dbLogger{db, tracer}
}

func parametersStringSlice(parameters ...any) []string {
	stringified := []string{}
	for _, parameter := range parameters {
		stringified = append(stringified, fmt.Sprintf("%v", parameter))
	}

	return stringified
}

func (d *dbLogger) recordSpan(ctx context.Context, spanName, query string, parameters ...any) (context.Context, trace.Span) {
	ctx, span := d.tracer.Start(ctx, spanName)
	span.SetAttributes(attribute.String("query", query))
	if len(parameters) > 0 {
		span.SetAttributes(attribute.StringSlice("parameters", parametersStringSlice(parameters)))
	}
	return ctx, span
}

func (d *dbLogger) ExecContext(ctx context.Context, query string, parameters ...any) (sql.Result, error) {
	logrus.Tracef("SQLITE: ExecContext(query=%#v, parameters=%+v)", query, parameters)

	ctx, span := d.recordSpan(ctx, "ExecContext", query, parameters...)
	defer span.End()

	result, err := d.db.ExecContext(ctx, query, parameters...)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return result, err
}

func (d *dbLogger) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	logrus.Tracef("SQLITE: PrepareContext(query=%#v)", query)

	ctx, span := d.recordSpan(ctx, "PrepareContext", query)
	defer span.End()

	result, err := d.db.PrepareContext(ctx, query)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return result, err
}

func (d *dbLogger) QueryContext(ctx context.Context, query string, parameters ...any) (*sql.Rows, error) {
	logrus.Tracef("SQLITE: QueryContext(query=%#v, parameters=%+v)", query, parameters)

	ctx, span := d.recordSpan(ctx, "QueryContext", query, parameters...)
	defer span.End()

	result, err := d.db.QueryContext(ctx, query, parameters...)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return result, err
}

func (d *dbLogger) QueryRowContext(ctx context.Context, query string, parameters ...any) *sql.Row {
	logrus.Tracef("SQLITE: QueryRowContext(query=%#v, parameters=%+v)", query, parameters)

	ctx, span := d.recordSpan(ctx, "QueryRowContext", query, parameters...)
	defer span.End()

	result := d.db.QueryRowContext(ctx, query, parameters...)
	if err := result.Err(); err != nil {
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return result
}
