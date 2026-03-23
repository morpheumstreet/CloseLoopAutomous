package geoip

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/oschwald/geoip2-golang"
	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

// NoopResolver never returns geo data.
type NoopResolver struct{}

var _ ports.GeoIPResolver = (*NoopResolver)(nil)

func (NoopResolver) LookupHost(_ context.Context, _ string) (*domain.GeoLocation, error) {
	return nil, nil
}

// MaxMindResolver uses a GeoLite2-City MMDB file (offline).
type MaxMindResolver struct {
	db *geoip2.Reader
}

var _ ports.GeoIPResolver = (*MaxMindResolver)(nil)

// OpenMaxMind opens path to a GeoIP2 or GeoLite2 City database. Caller must Close.
func OpenMaxMind(path string) (*MaxMindResolver, error) {
	db, err := geoip2.Open(path)
	if err != nil {
		return nil, err
	}
	return &MaxMindResolver{db: db}, nil
}

// Close releases the database handle.
func (r *MaxMindResolver) Close() error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.Close()
}

func (r *MaxMindResolver) LookupHost(ctx context.Context, host string) (*domain.GeoLocation, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	host = strings.TrimSpace(host)
	if host == "" || host == "localhost" {
		return noneGeo(), nil
	}
	if ip := net.ParseIP(host); ip != nil {
		return r.lookupIP(ip)
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil || len(ips) == 0 {
		return noneGeo(), nil
	}
	for _, ia := range ips {
		ip := ia.IP.To4()
		if ip == nil {
			ip = ia.IP
		}
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
			continue
		}
		return r.lookupIP(ip)
	}
	return noneGeo(), nil
}

func (r *MaxMindResolver) lookupIP(ip net.IP) (*domain.GeoLocation, error) {
	rec, err := r.db.City(ip)
	if err != nil {
		return noneGeo(), nil
	}
	now := time.Now().UTC()
	loc := &domain.GeoLocation{
		Latitude:    rec.Location.Latitude,
		Longitude:   rec.Location.Longitude,
		City:        rec.City.Names["en"],
		Region:      pickSubdivision(rec),
		Country:     rec.Country.Names["en"],
		CountryISO:  rec.Country.IsoCode,
		AccuracyKM:  int(rec.Location.AccuracyRadius),
		Source:      "maxmind_geoip2",
		LastUpdated: now,
	}
	if loc.CountryISO == "" && loc.Country == "" && loc.City == "" {
		return noneGeo(), nil
	}
	return loc, nil
}

func pickSubdivision(rec *geoip2.City) string {
	if len(rec.Subdivisions) == 0 {
		return ""
	}
	return rec.Subdivisions[0].Names["en"]
}

func noneGeo() *domain.GeoLocation {
	return &domain.GeoLocation{
		Source:      "none",
		LastUpdated: time.Now().UTC(),
	}
}
