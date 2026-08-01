package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/khanhnguyendev/mind-knowledge/internal/render"
	"github.com/khanhnguyendev/mind-knowledge/internal/store"
)

func init() {
	sourceCmd := &cobra.Command{
		Use:   "source",
		Short: "Capture and inspect raw sources",
		Long: "Sources are immutable. mk never fetches over the network: " +
			"pass --body, --file, --asset, or pipe the text on standard input.\n\n" +
			"Sources are cross-project, so no source command takes -p/--project.",
		PersistentPreRunE: crossProjectPreRun("sources"),
	}

	// add
	var addURI, addTitle, addKind, addBody, addFile, addAsset string
	var addForce bool
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Capture a source from --body, --file, --asset, or stdin",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := readSourceBody(addBody, addFile, addAsset)
			if err != nil {
				return err
			}
			if body == "" && addAsset == "" {
				return invalidf("no content: pass --body, --file, --asset, or pipe text on stdin")
			}

			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			if body != "" && !addForce {
				existing, err := s.FindSourceByHash(hashOf(body))
				if err == nil {
					return invalidf(
						"identical content already captured as source %s (%q); pass --force to store it again",
						existing.ID, existing.Title)
				}
				if !errors.Is(err, store.ErrNotFound) {
					return err
				}
			}

			src, err := s.CreateSource(addURI, addTitle, addKind, body, addAsset)
			if err != nil {
				return err
			}
			return emitCreated(src, src.ID)
		},
	}
	addCmd.Flags().StringVar(&addURI, "uri", "", "original address the content came from")
	addCmd.Flags().StringVar(&addTitle, "title", "", "source title (required)")
	addCmd.Flags().StringVar(&addKind, "kind", "note",
		"article, paper, transcript, chapter, asset, or note")
	addCmd.Flags().StringVar(&addBody, "body", "", "content as a literal string")
	addCmd.Flags().StringVar(&addFile, "file", "", "read content from this file")
	addCmd.Flags().StringVar(&addAsset, "asset", "",
		"path to an image or PDF held on disk instead of in the database")
	addCmd.Flags().BoolVar(&addForce, "force", false,
		"store even when identical content already exists")
	addCmd.MarkFlagRequired("title")

	// ls
	var lsKind string
	lsCmd := &cobra.Command{
		Use:   "ls",
		Short: "List sources, newest first",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			sources, err := s.ListSources(lsKind, LimitFlag())
			if err != nil {
				return err
			}
			if JSONMode() {
				return render.JSON(os.Stdout, sources)
			}
			rows := make([][]string, 0, len(sources))
			for _, src := range sources {
				rows = append(rows, []string{src.ID, src.Kind, src.Title, src.URI})
			}
			render.Table(os.Stdout, []string{"ID", "KIND", "TITLE", "URI"}, rows)
			return nil
		},
	}
	lsCmd.Flags().StringVar(&lsKind, "kind", "", "filter by kind")

	// view
	viewCmd := &cobra.Command{
		Use:   "view <id>",
		Short: "Show one source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			src, err := s.GetSource(args[0])
			if err != nil {
				return err
			}
			if JSONMode() {
				return render.JSON(os.Stdout, src)
			}
			fmt.Printf("%s  [%s]  %s\n", src.ID, src.Kind, src.Title)
			if src.URI != "" {
				fmt.Printf("uri: %s\n", src.URI)
			}
			if src.AssetPath != "" {
				fmt.Printf("asset: %s\n", src.AssetPath)
			}
			if src.Body != "" {
				fmt.Printf("\n%s\n", src.Body)
			}
			return nil
		},
	}

	// rm
	rmCmd := &cobra.Command{
		Use:   "rm <id>",
		Short: "Remove a source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			return s.DeleteSource(args[0])
		},
	}

	sourceCmd.AddCommand(addCmd, lsCmd, viewCmd, rmCmd)
	Root.AddCommand(sourceCmd)
}

// readSourceBody resolves content from --body, then --file, then stdin
// when stdin is a pipe rather than a terminal.
//
// asset short-circuits the stdin read exactly as body and file do. An
// asset-only source has its content on disk and wants no body, so falling
// through to io.ReadAll(os.Stdin) would block until whoever holds the
// other end of stdin closes it. The ModeCharDevice check does not save us:
// an agent harness hands its child an inherited pipe that nobody is
// writing to and nobody will close, which is a pipe by every test we can
// make and never yields a byte. mk would hang forever, silently.
func readSourceBody(body, file, asset string) (string, error) {
	if body != "" {
		return body, nil
	}
	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", invalidf("reading %s: %v", file, err)
		}
		return string(data), nil
	}
	if asset != "" {
		return "", nil
	}

	info, err := os.Stdin.Stat()
	if err != nil {
		// Report it rather than folding it into the silent "no content"
		// path: the caller passed no content flag and we cannot tell
		// whether stdin holds any, so "no content" would be a guess.
		return "", invalidf("checking stdin: %v", err)
	}
	if info.Mode()&os.ModeCharDevice != 0 {
		// stdin is a terminal, so there is nothing piped in.
		return "", nil
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", invalidf("reading stdin: %v", err)
	}
	return string(data), nil
}

// hashOf mirrors the store's content hashing so add can look for a
// duplicate before inserting.
func hashOf(body string) string { return store.HashBody(body) }
