package shipping

import (
	"testing"
)

func TestNewPullRequestPublisher_backend(t *testing.T) {
	t.Parallel()
	if _, ok := NewPullRequestPublisher(PublisherSettings{PRBackend: "gh"}).(*GhCLIPublisher); !ok {
		t.Fatal("want GhCLIPublisher for backend gh")
	}
	if _, ok := NewPullRequestPublisher(PublisherSettings{PRBackend: "gh-cli"}).(*GhCLIPublisher); !ok {
		t.Fatal("want GhCLIPublisher for backend gh-cli")
	}
	if _, ok := NewPullRequestPublisher(PublisherSettings{PRBackend: "", APIToken: "tok"}).(*GitHubPublisher); !ok {
		t.Fatal("want GitHubPublisher when token set and default backend")
	}
	if _, ok := NewPullRequestPublisher(PublisherSettings{PRBackend: "api", APIToken: "tok"}).(*GitHubPublisher); !ok {
		t.Fatal("want GitHubPublisher for explicit api backend")
	}
	var noop PullRequestNoop
	if _, ok := NewPullRequestPublisher(PublisherSettings{}).(PullRequestNoop); !ok {
		t.Fatalf("want noop without token, got %T", NewPullRequestPublisher(PublisherSettings{}))
	}
	if _, ok := NewPullRequestPublisher(PublisherSettings{PRBackend: "api"}).(PullRequestNoop); !ok {
		t.Fatal("want noop for api without token")
	}
	if _, ok := NewPullRequestPublisher(PublisherSettings{PRBackend: "bogus"}).(PullRequestNoop); !ok {
		t.Fatal("want noop for unknown backend")
	}
	_ = noop
}
