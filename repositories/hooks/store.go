package hooks

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"

	"github.com/reviewboard/rb-gateway/repositories/events"
)

// A collection of webhooks, mapped to by their `Id`.
type WebhookStore map[string]*Webhook

// Load a collection of webhooks from the given reader.
//
// The store is expected to be unmarshalled from JSON.
//
// `repositories` must be a set of all repository names.
//
// If a webhook references a non-extant repository, that repository will be
// stripped from the loaded webhook. Likewise, if a webhook references an
// invalid event that too will be stripped.
//
// As a side effect, the `Events` and `Repos` fields of each hook will be
// sorted.
func LoadStore(path string, repositories map[string]struct{}) (WebhookStore, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(WebhookStore), nil
		}

		return nil, err
	}
	defer f.Close()

	store, err := ReadStore(f, repositories)
	if err != nil {
		if err, ok := err.(*json.SyntaxError); ok && err.Offset == 0 && err.Error() == "unexpected end of JSON input" {
			// The file is empty, so return an empty store.
			return make(WebhookStore), nil
		}
	}

	return store, err
}

// Read a collection of webhooks from the given reader.
//
// Callers should prefer the higher-level `LoadStore` over this function.
func ReadStore(r io.Reader, repositories map[string]struct{}) (WebhookStore, error) {
	content, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	rawStore := []*Webhook{}
	if err = json.Unmarshal(content, &rawStore); err != nil {
		return nil, err
	}

	store := make(WebhookStore)

	for _, hook := range rawStore {
		if validateHook(hook, repositories) {
			store[hook.Id] = hook
		}
	}

	return store, nil
}

// Save the WebhookStore.
//
// The store will first be written to a temporary file and will then be moved
// to the target location. This is done to avoid `rb-gateway trigger-webhooks`
// processess from reading the file as we are writing to it, causing errors.
func (s WebhookStore) Save(path string) error {
	tmpfile, err := ioutil.TempFile("", "tmp-store")
	if err != nil {
		return err
	}

	err = s.Write(tmpfile)
	tmpfile.Close()

	if err != nil {
		return err
	}

	return os.Rename(tmpfile.Name(), path)
}

// Write the store to a writer.
//
// The store will be marshalled as JSON.
//
// Callers should prefer the higher-level `WebhookStore.Save` over this function.
func (s WebhookStore) Write(w io.Writer) error {
	rawStore := make([]Webhook, 0, len(s))

	for _, hook := range s {
		rawStore = append(rawStore, *hook)
	}

	content, err := json.MarshalIndent(rawStore, "", "  ")
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(content))
	return err
}

// Validate a hook, stripping invalid fields.
//
// If an invalid event or repository is specified, it will be stripped from the
// hook.
//
// As a side effect, the `Events` and `Repos` fields of each hook will be
// sorted.
func validateHook(hook *Webhook, repos map[string]struct{}) bool {
	validEvents := make([]string, 0, len(hook.Events))
	validRepos := make([]string, 0, len(hook.Repos))

	for _, event := range hook.Events {
		if events.IsValidEvent(event) {
			validEvents = append(validEvents, event)
		} else {
			log.Printf(`Unknown event type "%s" in hook "%s"; skipping event.`,
				event, hook.Id)
		}
	}

	for _, repo := range hook.Repos {
		if _, ok := repos[repo]; ok {
			validRepos = append(validRepos, repo)
		} else {
			log.Printf(`Unknown repo "%s" in hook "%s"; skipping event.`,
				repo, hook.Id)
		}
	}

	if len(validEvents) == 0 {
		log.Printf(`Hook "%s" has no valid events; skipping hook.`, hook.Id)
		return false
	} else if len(validRepos) == 0 {
		log.Printf(`Hook "%s" has no valid repositories; skipping hook.`, hook.Id)
		return false
	}

	if len(hook.Secret) < 20 {
		log.Printf(
			`WARNING: Secret for webhook "%s" is too short (%d bytes); should be at least 20 bytes.`,
			hook.Id, len(hook.Secret))
	}

	sort.Strings(validEvents)
	hook.Events = validEvents

	sort.Strings(validRepos)
	hook.Repos = validRepos

	return true
}

// Iterate over all the webhooks that match the specified event and repository.
//
// `f` will be called for each repository. Errors will not stop iteration from
// continuing. `f` will only be called for webhooks that are enabled for the
// given event and repository name.
//
// All errors will be returned as a slice (which will be `nil` if there were no errors).
func (store WebhookStore) ForEach(event, repoName string, f func(h Webhook) error) []error {
	errs := []error{}

	for _, hook := range store {
		if hook.Enabled && contains(hook.Repos, repoName) && contains(hook.Events, event) {
			err := f(*hook)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) != 0 {
		return errs
	} else {
		return nil
	}
}

// Check if `haystack` contains `needle`.
func contains(haystack []string, needle string) bool {
	index := sort.SearchStrings(haystack, needle)
	return index != len(haystack) && haystack[index] == needle
}
