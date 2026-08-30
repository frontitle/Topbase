package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/topbase/topbase/internal/core"
	"github.com/topbase/topbase/internal/core/queryir"
	"github.com/topbase/topbase/internal/topbaseapi"
)

func main() {
	baseURL := flag.String("url", envOr("TOPBASE_URL", "http://localhost"), "Topbase base URL")
	apiKey := flag.String("api-key", os.Getenv("TOPBASE_API_KEY"), "Topbase API key (prefer TOPBASE_API_KEY)")
	maxRows := flag.Int("max-rows", 200, "maximum rows returned per query")
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() == 0 {
		usage()
		os.Exit(2)
	}
	client, err := topbaseapi.New(*baseURL, *apiKey, topbaseapi.WithMaxRows(*maxRows))
	if err != nil {
		fail(err)
	}
	if err := run(context.Background(), client, flag.Arg(0), flag.Args()[1:]); err != nil {
		fail(err)
	}
}

func run(ctx context.Context, client *topbaseapi.Client, command string, args []string) error {
	var output any
	var err error
	switch command {
	case "status":
		output, err = client.Status(ctx)
	case "databases":
		output, err = client.ListDatabases(ctx)
	case "tables":
		if len(args) != 1 {
			return errors.New("usage: topbase-cli tables DATABASE_ID")
		}
		output, err = client.ListTables(ctx, args[0])
	case "describe":
		if len(args) != 3 {
			return errors.New("usage: topbase-cli describe DATABASE_ID SCHEMA TABLE")
		}
		output, err = client.DescribeTable(ctx, args[0], args[1], args[2])
	case "groups":
		output, err = client.ListCollections(ctx)
	case "analyses":
		output, err = client.ListAnalyses(ctx)
	case "query":
		set := flag.NewFlagSet("query", flag.ContinueOnError)
		file := set.String("file", "-", "QueryIR JSON file, or - for stdin")
		if err := set.Parse(args); err != nil {
			return err
		}
		query, readErr := readQuery(*file)
		if readErr != nil {
			return readErr
		}
		output, err = client.Preview(ctx, query)
	case "run-analysis":
		if len(args) != 1 {
			return errors.New("usage: topbase-cli run-analysis ANALYSIS_ID")
		}
		output, err = client.RunAnalysis(ctx, args[0])
	case "create-analysis":
		set := flag.NewFlagSet("create-analysis", flag.ContinueOnError)
		name := set.String("name", "", "analysis name")
		description := set.String("description", "", "analysis description")
		group := set.String("group", "", "destination data group ID; defaults to personal group")
		file := set.String("file", "-", "QueryIR JSON file, or - for stdin")
		confirmed := set.Bool("confirm", false, "confirm the user requested this write")
		if err := set.Parse(args); err != nil {
			return err
		}
		if !*confirmed {
			return errors.New("--confirm is required because creating an analysis changes Topbase")
		}
		query, readErr := readQuery(*file)
		if readErr != nil {
			return readErr
		}
		output, err = client.CreateAnalysis(ctx, core.Question{
			Name: *name, Description: *description, CollectionID: *group, QueryIR: &query,
		})
	default:
		return fmt.Errorf("unknown command %q", command)
	}
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func readQuery(filename string) (queryir.Query, error) {
	var reader io.Reader = os.Stdin
	if filename != "-" {
		file, err := os.Open(filename)
		if err != nil {
			return queryir.Query{}, err
		}
		defer file.Close()
		reader = file
	}
	var query queryir.Query
	if err := json.NewDecoder(io.LimitReader(reader, 1<<20)).Decode(&query); err != nil {
		return queryir.Query{}, fmt.Errorf("read QueryIR: %w", err)
	}
	return query, nil
}

func usage() {
	fmt.Fprintln(flag.CommandLine.Output(), `Usage: topbase-cli [global flags] COMMAND [arguments]

Commands:
  status                                 Check URL, API key and server limits
  databases                              List visible source databases
  tables DATABASE_ID                     List live tables and comments
  describe DATABASE_ID SCHEMA TABLE      Read columns and semantic metadata
  groups                                 List visible analysis data groups
  analyses                               List visible saved analyses
  query --file query.json                Run bounded, read-only QueryIR
  run-analysis ANALYSIS_ID               Run a saved visual analysis
  create-analysis --name NAME --file query.json --confirm

Global flags use TOPBASE_URL and TOPBASE_API_KEY by default.`)
	flag.PrintDefaults()
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "topbase-cli:", err)
	os.Exit(1)
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
