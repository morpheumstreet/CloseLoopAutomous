package gateway

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/closeloopautomous/arms/internal/adapters/gateway/clawlet"
	"github.com/closeloopautomous/arms/internal/adapters/gateway/copaw"
	"github.com/closeloopautomous/arms/internal/adapters/gateway/inkos"
	"github.com/closeloopautomous/arms/internal/adapters/gateway/ironclaw"
	"github.com/closeloopautomous/arms/internal/adapters/gateway/metaclaw"
	"github.com/closeloopautomous/arms/internal/adapters/gateway/mimiclaw"
	"github.com/closeloopautomous/arms/internal/adapters/gateway/mistermorph"
	"github.com/closeloopautomous/arms/internal/adapters/gateway/nanobotcli"
	"github.com/closeloopautomous/arms/internal/adapters/gateway/nemoclaw"
	"github.com/closeloopautomous/arms/internal/adapters/gateway/nullclaw"
	"github.com/closeloopautomous/arms/internal/adapters/gateway/openclaw"
	"github.com/closeloopautomous/arms/internal/adapters/gateway/picoclaw"
	"github.com/closeloopautomous/arms/internal/adapters/gateway/zclaw"
	"github.com/closeloopautomous/arms/internal/adapters/gateway/zeroclaw"
	"github.com/closeloopautomous/arms/internal/domain"
)

// clientPool reuses gateway clients per (driver, url, token, device, timeout).
type clientPool struct {
	mu             sync.Mutex
	openclaw       map[string]*openclaw.Client
	zeroclaw       map[string]*zeroclaw.Client
	clawlet        map[string]*clawlet.Client
	ironclaw       map[string]*ironclaw.Client
	picoclaw       map[string]*picoclaw.Client
	mimiclaw       map[string]*mimiclaw.Client
	nanobotCLI     map[string]*nanobotcli.Client
	inkosCLI       map[string]*inkos.Client
	nullclawHTTP   map[string]*nullclaw.Client
	zclawRelay     map[string]*zclaw.Client
	misterMorph    map[string]*mistermorph.Client
	copawHTTP      map[string]*copaw.Client
	metaclawHTTP   map[string]*metaclaw.Client
	nemoclawWS     map[string]*nemoclaw.Client
	nemo           nemoclaw.PoolSettings
	knowledge      func(context.Context, domain.ProductID, string) (string, error)
	defaultTimeout time.Duration
}

func newClientPool(knowledge func(context.Context, domain.ProductID, string) (string, error), defaultTimeout time.Duration, nemo nemoclaw.PoolSettings) *clientPool {
	return &clientPool{
		openclaw:       make(map[string]*openclaw.Client),
		zeroclaw:       make(map[string]*zeroclaw.Client),
		clawlet:        make(map[string]*clawlet.Client),
		ironclaw:       make(map[string]*ironclaw.Client),
		picoclaw:       make(map[string]*picoclaw.Client),
		mimiclaw:       make(map[string]*mimiclaw.Client),
		nanobotCLI:     make(map[string]*nanobotcli.Client),
		inkosCLI:       make(map[string]*inkos.Client),
		nullclawHTTP:   make(map[string]*nullclaw.Client),
		zclawRelay:     make(map[string]*zclaw.Client),
		misterMorph:    make(map[string]*mistermorph.Client),
		copawHTTP:      make(map[string]*copaw.Client),
		metaclawHTTP:   make(map[string]*metaclaw.Client),
		nemoclawWS:     make(map[string]*nemoclaw.Client),
		nemo:           nemo,
		knowledge:      knowledge,
		defaultTimeout: defaultTimeout,
	}
}

func (p *clientPool) key(target domain.DispatchTarget) string {
	to := target.Timeout
	if to <= 0 {
		to = p.defaultTimeout
	}
	if to <= 0 {
		to = 30 * time.Second
	}
	// NemoClaw pool key must include process-wide CLI settings so toggling ARMS_NEMOCLAW_* does not reuse stale clients.
	if target.Driver == domain.GatewayDriverNemoClawWS {
		return fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s\n%v\n%s", target.Driver, target.GatewayURL, target.GatewayToken, target.DeviceID, to.String(),
			p.nemo.BinaryPath, p.nemo.AutoStart, p.nemo.DefaultBlueprint)
	}
	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s", target.Driver, target.GatewayURL, target.GatewayToken, target.DeviceID, to.String())
}

func (p *clientPool) openclawClientFor(target domain.DispatchTarget) *openclaw.Client {
	to := target.Timeout
	if to <= 0 {
		to = p.defaultTimeout
	}
	if to <= 0 {
		to = 30 * time.Second
	}
	k := p.key(target)
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.openclaw[k]; ok {
		return c
	}
	opts := openclaw.Options{
		URL:                  target.GatewayURL,
		Token:                target.GatewayToken,
		DeviceID:             target.DeviceID,
		SessionKey:           "",
		Timeout:              to,
		KnowledgeForDispatch: p.knowledge,
	}
	c := openclaw.New(opts)
	p.openclaw[k] = c
	return c
}

func (p *clientPool) nemoclawClientFor(target domain.DispatchTarget) *nemoclaw.Client {
	to := target.Timeout
	if to <= 0 {
		to = p.defaultTimeout
	}
	if to <= 0 {
		to = 30 * time.Second
	}
	k := p.key(target)
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.nemoclawWS[k]; ok {
		return c
	}
	c := nemoclaw.New(nemoclaw.Options{
		URL:                  target.GatewayURL,
		Token:                target.GatewayToken,
		DeviceID:             target.DeviceID,
		Timeout:              to,
		SandboxName:          target.DeviceID,
		NemoClawBin:          p.nemo.BinaryPath,
		AutoStart:            p.nemo.AutoStart,
		KnowledgeForDispatch: p.knowledge,
	})
	p.nemoclawWS[k] = c
	return c
}

func (p *clientPool) zeroclawClientFor(target domain.DispatchTarget) *zeroclaw.Client {
	to := target.Timeout
	if to <= 0 {
		to = p.defaultTimeout
	}
	if to <= 0 {
		to = 30 * time.Second
	}
	k := p.key(target)
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.zeroclaw[k]; ok {
		return c
	}
	c := zeroclaw.New(zeroclaw.Options{
		URL:                  target.GatewayURL,
		Token:                target.GatewayToken,
		DeviceID:             target.DeviceID,
		Timeout:              to,
		KnowledgeForDispatch: p.knowledge,
	})
	p.zeroclaw[k] = c
	return c
}

func (p *clientPool) clawletClientFor(target domain.DispatchTarget) *clawlet.Client {
	to := target.Timeout
	if to <= 0 {
		to = p.defaultTimeout
	}
	if to <= 0 {
		to = 30 * time.Second
	}
	k := p.key(target)
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.clawlet[k]; ok {
		return c
	}
	c := clawlet.New(clawlet.Options{
		URL:                  target.GatewayURL,
		Token:                target.GatewayToken,
		DeviceID:             target.DeviceID,
		Timeout:              to,
		KnowledgeForDispatch: p.knowledge,
	})
	p.clawlet[k] = c
	return c
}

func (p *clientPool) ironclawClientFor(target domain.DispatchTarget) *ironclaw.Client {
	to := target.Timeout
	if to <= 0 {
		to = p.defaultTimeout
	}
	if to <= 0 {
		to = 30 * time.Second
	}
	k := p.key(target)
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.ironclaw[k]; ok {
		return c
	}
	c := ironclaw.New(ironclaw.Options{
		URL:                  target.GatewayURL,
		Token:                target.GatewayToken,
		DeviceID:             target.DeviceID,
		Timeout:              to,
		KnowledgeForDispatch: p.knowledge,
	})
	p.ironclaw[k] = c
	return c
}

func (p *clientPool) picoclawClientFor(target domain.DispatchTarget) *picoclaw.Client {
	to := target.Timeout
	if to <= 0 {
		to = p.defaultTimeout
	}
	if to <= 0 {
		to = 30 * time.Second
	}
	k := p.key(target)
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.picoclaw[k]; ok {
		return c
	}
	c := picoclaw.New(picoclaw.Options{
		URL:                  target.GatewayURL,
		Token:                target.GatewayToken,
		Timeout:              to,
		KnowledgeForDispatch: p.knowledge,
	})
	p.picoclaw[k] = c
	return c
}

func (p *clientPool) mimiclawClientFor(target domain.DispatchTarget) *mimiclaw.Client {
	to := target.Timeout
	if to <= 0 {
		to = p.defaultTimeout
	}
	if to <= 0 {
		to = 30 * time.Second
	}
	k := p.key(target)
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.mimiclaw[k]; ok {
		return c
	}
	c := mimiclaw.New(mimiclaw.Options{
		URL:                  target.GatewayURL,
		Token:                target.GatewayToken,
		Timeout:              to,
		KnowledgeForDispatch: p.knowledge,
	})
	p.mimiclaw[k] = c
	return c
}

func (p *clientPool) nullclawClientFor(target domain.DispatchTarget) *nullclaw.Client {
	to := target.Timeout
	if to <= 0 {
		to = p.defaultTimeout
	}
	if to <= 0 {
		to = 30 * time.Second
	}
	k := p.key(target)
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.nullclawHTTP[k]; ok {
		return c
	}
	c := nullclaw.New(nullclaw.Options{
		BaseURL:              target.GatewayURL,
		Token:                target.GatewayToken,
		Timeout:              to,
		KnowledgeForDispatch: p.knowledge,
	})
	p.nullclawHTTP[k] = c
	return c
}

func (p *clientPool) zclawClientFor(target domain.DispatchTarget) *zclaw.Client {
	to := target.Timeout
	if to <= 0 {
		to = p.defaultTimeout
	}
	if to <= 0 {
		to = 30 * time.Second
	}
	k := p.key(target)
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.zclawRelay[k]; ok {
		return c
	}
	c := zclaw.New(zclaw.Options{
		BaseURL:              target.GatewayURL,
		Token:                target.GatewayToken,
		Timeout:              to,
		KnowledgeForDispatch: p.knowledge,
	})
	p.zclawRelay[k] = c
	return c
}

func (p *clientPool) inkosCLIClientFor(target domain.DispatchTarget) *inkos.Client {
	to := target.Timeout
	if to <= 0 {
		to = p.defaultTimeout
	}
	if to <= 0 {
		to = 30 * time.Second
	}
	k := p.key(target)
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.inkosCLI[k]; ok {
		return c
	}
	c := inkos.New(inkos.Options{
		InkOSBin:             target.GatewayToken,
		Workspace:            target.GatewayURL,
		BookID:               target.DeviceID,
		Timeout:              to,
		KnowledgeForDispatch: p.knowledge,
	})
	p.inkosCLI[k] = c
	return c
}

func (p *clientPool) nanobotCLIClientFor(target domain.DispatchTarget) *nanobotcli.Client {
	to := target.Timeout
	if to <= 0 {
		to = p.defaultTimeout
	}
	if to <= 0 {
		to = 30 * time.Second
	}
	k := p.key(target)
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.nanobotCLI[k]; ok {
		return c
	}
	c := nanobotcli.New(nanobotcli.Options{
		NanobotBin:           target.GatewayToken,
		ConfigPath:           target.GatewayURL,
		Workspace:            target.DeviceID,
		Timeout:              to,
		KnowledgeForDispatch: p.knowledge,
	})
	p.nanobotCLI[k] = c
	return c
}

func (p *clientPool) mistermorphClientFor(target domain.DispatchTarget) *mistermorph.Client {
	to := target.Timeout
	if to <= 0 {
		to = p.defaultTimeout
	}
	if to <= 0 {
		to = 30 * time.Second
	}
	k := p.key(target)
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.misterMorph[k]; ok {
		return c
	}
	c := mistermorph.New(mistermorph.Options{
		BaseURL:              target.GatewayURL,
		Token:                target.GatewayToken,
		ModelOverride:        target.DeviceID,
		Timeout:              to,
		KnowledgeForDispatch: p.knowledge,
	})
	p.misterMorph[k] = c
	return c
}

func (p *clientPool) copawClientFor(target domain.DispatchTarget) *copaw.Client {
	to := target.Timeout
	if to <= 0 {
		to = p.defaultTimeout
	}
	if to <= 0 {
		to = 30 * time.Second
	}
	k := p.key(target)
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.copawHTTP[k]; ok {
		return c
	}
	c := copaw.New(copaw.Options{
		BaseURL:              target.GatewayURL,
		Token:                target.GatewayToken,
		Workspace:            target.DeviceID,
		Timeout:              to,
		KnowledgeForDispatch: p.knowledge,
	})
	p.copawHTTP[k] = c
	return c
}

func (p *clientPool) metaclawClientFor(target domain.DispatchTarget) *metaclaw.Client {
	to := target.Timeout
	if to <= 0 {
		to = p.defaultTimeout
	}
	if to <= 0 {
		to = 30 * time.Second
	}
	k := p.key(target)
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.metaclawHTTP[k]; ok {
		return c
	}
	c := metaclaw.New(metaclaw.Options{
		BaseURL:              target.GatewayURL,
		Token:                target.GatewayToken,
		ModelOverride:        target.DeviceID,
		Timeout:              to,
		KnowledgeForDispatch: p.knowledge,
	})
	p.metaclawHTTP[k] = c
	return c
}

func (p *clientPool) dispatchTask(ctx context.Context, target domain.DispatchTarget, task domain.Task) (string, error) {
	switch target.Driver {
	case domain.GatewayDriverPicoClawWS:
		return p.picoclawClientFor(target).DispatchTaskWithSession(ctx, task, target.SessionKey)
	case domain.GatewayDriverMimiClawWS:
		return p.mimiclawClientFor(target).DispatchTaskWithSession(ctx, task, target.SessionKey)
	case domain.GatewayDriverNullClawA2A:
		return p.nullclawClientFor(target).DispatchTaskWithSession(ctx, task, target.SessionKey)
	case domain.GatewayDriverZeroClawWS:
		return p.zeroclawClientFor(target).DispatchTaskWithSession(ctx, task, target.SessionKey)
	case domain.GatewayDriverClawletWS:
		return p.clawletClientFor(target).DispatchTaskWithSession(ctx, task, target.SessionKey)
	case domain.GatewayDriverIronClawWS:
		return p.ironclawClientFor(target).DispatchTaskWithSession(ctx, task, target.SessionKey)
	case domain.GatewayDriverNanobotCLI:
		return p.nanobotCLIClientFor(target).DispatchTaskWithSession(ctx, task, target.SessionKey)
	case domain.GatewayDriverInkOSCLI:
		return p.inkosCLIClientFor(target).DispatchTaskWithSession(ctx, task, target.SessionKey)
	case domain.GatewayDriverZClawRelayHTTP:
		return p.zclawClientFor(target).DispatchTaskWithSession(ctx, task, target.SessionKey)
	case domain.GatewayDriverMisterMorphHTTP:
		return p.mistermorphClientFor(target).DispatchTaskWithSession(ctx, task, target.SessionKey)
	case domain.GatewayDriverCoPawHTTP:
		return p.copawClientFor(target).DispatchTaskWithSession(ctx, task, target.SessionKey)
	case domain.GatewayDriverMetaClawHTTP:
		return p.metaclawClientFor(target).DispatchTaskWithSession(ctx, task, target.SessionKey)
	case domain.GatewayDriverNemoClawWS:
		return p.nemoclawClientFor(target).DispatchTaskWithSession(ctx, task, target.SessionKey)
	default:
		return p.openclawClientFor(target).DispatchTaskWithSession(ctx, task, target.SessionKey)
	}
}

func (p *clientPool) dispatchSubtask(ctx context.Context, target domain.DispatchTarget, parent domain.Task, sub domain.Subtask) (string, error) {
	switch target.Driver {
	case domain.GatewayDriverPicoClawWS:
		return p.picoclawClientFor(target).DispatchSubtaskWithSession(ctx, parent, sub, target.SessionKey)
	case domain.GatewayDriverMimiClawWS:
		return p.mimiclawClientFor(target).DispatchSubtaskWithSession(ctx, parent, sub, target.SessionKey)
	case domain.GatewayDriverNullClawA2A:
		return p.nullclawClientFor(target).DispatchSubtaskWithSession(ctx, parent, sub, target.SessionKey)
	case domain.GatewayDriverZeroClawWS:
		return p.zeroclawClientFor(target).DispatchSubtaskWithSession(ctx, parent, sub, target.SessionKey)
	case domain.GatewayDriverClawletWS:
		return p.clawletClientFor(target).DispatchSubtaskWithSession(ctx, parent, sub, target.SessionKey)
	case domain.GatewayDriverIronClawWS:
		return p.ironclawClientFor(target).DispatchSubtaskWithSession(ctx, parent, sub, target.SessionKey)
	case domain.GatewayDriverNanobotCLI:
		return p.nanobotCLIClientFor(target).DispatchSubtaskWithSession(ctx, parent, sub, target.SessionKey)
	case domain.GatewayDriverInkOSCLI:
		return p.inkosCLIClientFor(target).DispatchSubtaskWithSession(ctx, parent, sub, target.SessionKey)
	case domain.GatewayDriverZClawRelayHTTP:
		return p.zclawClientFor(target).DispatchSubtaskWithSession(ctx, parent, sub, target.SessionKey)
	case domain.GatewayDriverMisterMorphHTTP:
		return p.mistermorphClientFor(target).DispatchSubtaskWithSession(ctx, parent, sub, target.SessionKey)
	case domain.GatewayDriverCoPawHTTP:
		return p.copawClientFor(target).DispatchSubtaskWithSession(ctx, parent, sub, target.SessionKey)
	case domain.GatewayDriverMetaClawHTTP:
		return p.metaclawClientFor(target).DispatchSubtaskWithSession(ctx, parent, sub, target.SessionKey)
	case domain.GatewayDriverNemoClawWS:
		return p.nemoclawClientFor(target).DispatchSubtaskWithSession(ctx, parent, sub, target.SessionKey)
	default:
		return p.openclawClientFor(target).DispatchSubtaskWithSession(ctx, parent, sub, target.SessionKey)
	}
}

func (p *clientPool) close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range p.openclaw {
		_ = c.Close()
	}
	for _, c := range p.zeroclaw {
		_ = c.Close()
	}
	for _, c := range p.clawlet {
		_ = c.Close()
	}
	for _, c := range p.ironclaw {
		_ = c.Close()
	}
	for _, c := range p.picoclaw {
		_ = c.Close()
	}
	for _, c := range p.mimiclaw {
		_ = c.Close()
	}
	for _, c := range p.nanobotCLI {
		_ = c.Close()
	}
	for _, c := range p.inkosCLI {
		_ = c.Close()
	}
	for _, c := range p.nullclawHTTP {
		_ = c.Close()
	}
	for _, c := range p.zclawRelay {
		_ = c.Close()
	}
	for _, c := range p.misterMorph {
		_ = c.Close()
	}
	for _, c := range p.copawHTTP {
		_ = c.Close()
	}
	for _, c := range p.metaclawHTTP {
		_ = c.Close()
	}
	for _, c := range p.nemoclawWS {
		_ = c.Close()
	}
	p.openclaw = make(map[string]*openclaw.Client)
	p.zeroclaw = make(map[string]*zeroclaw.Client)
	p.clawlet = make(map[string]*clawlet.Client)
	p.ironclaw = make(map[string]*ironclaw.Client)
	p.picoclaw = make(map[string]*picoclaw.Client)
	p.mimiclaw = make(map[string]*mimiclaw.Client)
	p.nanobotCLI = make(map[string]*nanobotcli.Client)
	p.inkosCLI = make(map[string]*inkos.Client)
	p.nullclawHTTP = make(map[string]*nullclaw.Client)
	p.zclawRelay = make(map[string]*zclaw.Client)
	p.misterMorph = make(map[string]*mistermorph.Client)
	p.copawHTTP = make(map[string]*copaw.Client)
	p.metaclawHTTP = make(map[string]*metaclaw.Client)
	p.nemoclawWS = make(map[string]*nemoclaw.Client)
}
