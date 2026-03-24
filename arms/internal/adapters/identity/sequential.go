package identity

import (
	"fmt"
	"sync"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/ports"
)

// Sequential issues deterministic IDs for tests and local runs.
type Sequential struct {
	mu sync.Mutex
	n  int
}

func (s *Sequential) NewProductID() domain.ProductID { return domain.ProductID(s.next("prod")) }

func (s *Sequential) next(prefix string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.n++
	return fmt.Sprintf("%s-%d", prefix, s.n)
}

func (s *Sequential) NewIdeaID() domain.IdeaID     { return domain.IdeaID(s.next("idea")) }
func (s *Sequential) NewTaskID() domain.TaskID     { return domain.TaskID(s.next("task")) }
func (s *Sequential) NewConvoyID() domain.ConvoyID { return domain.ConvoyID(s.next("convoy")) }
func (s *Sequential) NewSubtaskID() domain.SubtaskID {
	return domain.SubtaskID(s.next("sub"))
}
func (s *Sequential) NewCostEventID() string { return s.next("cost") }

func (s *Sequential) NewResearchCycleID() string { return s.next("rc") }

func (s *Sequential) NewExecutionAgentID() string { return s.next("agent") }

func (s *Sequential) NewMailboxMessageID() string { return s.next("mail") }

func (s *Sequential) NewProductFeedbackID() string { return s.next("fb") }

func (s *Sequential) NewTaskChatMessageID() string { return s.next("tchat") }

func (s *Sequential) NewGatewayEndpointID() string { return s.next("gw") }

var _ ports.IdentityGenerator = (*Sequential)(nil)
