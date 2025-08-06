package ipinfo

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// IPInfo contains detailed information about an IP address
type IPInfo struct {
	IP          string  `json:"ip"`
	Country     string  `json:"country,omitempty"`
	CountryCode string  `json:"country_code,omitempty"`
	Region      string  `json:"region,omitempty"`
	City        string  `json:"city,omitempty"`
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
	ISP         string  `json:"isp,omitempty"`
	ASN         string  `json:"asn,omitempty"`
	Timezone    string  `json:"timezone,omitempty"`
	IsPrivate   bool    `json:"is_private"`
	IsLoopback  bool    `json:"is_loopback"`
	CacheTime   time.Time `json:"cache_time"`
}

// Service provides IP geolocation functionality
type Service struct {
	cache       map[string]*IPInfo
	cacheMu     sync.RWMutex
	maxAge      time.Duration
	maxSize     int
	client      *http.Client
	lastCleanup time.Time
}

// NewService creates a new IP geolocation service
func NewService() *Service {
	return &Service{
		cache:       make(map[string]*IPInfo),
		maxAge:      24 * time.Hour, // Cache for 24 hours
		maxSize:     1000,           // Maximum cache entries
		client:      &http.Client{
			Timeout: 5 * time.Second,
		},
		lastCleanup: time.Now(),
	}
}

// GetIPInfo retrieves detailed information about an IP address
func (s *Service) GetIPInfo(ip string) *IPInfo {
	// Parse the IP to get additional metadata
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return &IPInfo{
			IP:        ip,
			IsPrivate: false,
			CacheTime: time.Now(),
		}
	}

	// Check if IP is private, loopback, etc.
	isPrivate := parsedIP.IsPrivate()
	isLoopback := parsedIP.IsLoopback()

	// Check cache first
	s.cacheMu.RLock()
	if cached, exists := s.cache[ip]; exists {
		if time.Since(cached.CacheTime) < s.maxAge {
			s.cacheMu.RUnlock()
			return cached
		}
	}
	s.cacheMu.RUnlock()

	// Create base info
	info := &IPInfo{
		IP:         ip,
		IsPrivate:  isPrivate,
		IsLoopback: isLoopback,
		CacheTime:  time.Now(),
	}

	// Skip geolocation for private/loopback IPs
	if isPrivate || isLoopback {
		if isLoopback {
			info.Country = "Local"
			info.CountryCode = "LO"
			info.City = "Localhost"
		} else {
			info.Country = "Private Network"
			info.CountryCode = "PR"
		}
		s.cacheIPInfo(ip, info)
		return info
	}

	// Try to get geolocation from external service
	s.enrichWithGeolocation(info)
	
	// Cache the result
	s.cacheIPInfo(ip, info)
	
	return info
}

// enrichWithGeolocation fetches geolocation data from external API
func (s *Service) enrichWithGeolocation(info *IPInfo) {
	// Using ipapi.co as it's free and doesn't require API keys for basic usage
	url := fmt.Sprintf("https://ipapi.co/%s/json/", info.IP)
	
	resp, err := s.client.Get(url)
	if err != nil {
		return // Fail silently, we'll have basic IP info
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	var geoData struct {
		Country     string  `json:"country_name"`
		CountryCode string  `json:"country_code"`
		Region      string  `json:"region"`
		City        string  `json:"city"`
		Latitude    float64 `json:"latitude"`
		Longitude   float64 `json:"longitude"`
		ISP         string  `json:"org"`
		ASN         string  `json:"asn"`
		Timezone    string  `json:"timezone"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&geoData); err != nil {
		return // Fail silently
	}

	// Update the info with geolocation data
	info.Country = geoData.Country
	info.CountryCode = geoData.CountryCode
	info.Region = geoData.Region
	info.City = geoData.City
	info.Latitude = geoData.Latitude
	info.Longitude = geoData.Longitude
	info.ISP = geoData.ISP
	info.ASN = geoData.ASN
	info.Timezone = geoData.Timezone
}

// cacheIPInfo stores IP info in cache with size management
func (s *Service) cacheIPInfo(ip string, info *IPInfo) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	
	// Check if we need to clean up expired entries
	if time.Since(s.lastCleanup) > time.Hour {
		s.cleanupExpiredEntries()
		s.lastCleanup = time.Now()
	}
	
	// If cache is at max size and this is a new entry, remove oldest
	if len(s.cache) >= s.maxSize {
		if _, exists := s.cache[ip]; !exists {
			s.evictOldestEntry()
		}
	}
	
	s.cache[ip] = info
}

// cleanupExpiredEntries removes expired entries from cache
func (s *Service) cleanupExpiredEntries() {
	now := time.Now()
	for ip, info := range s.cache {
		if now.Sub(info.CacheTime) > s.maxAge {
			delete(s.cache, ip)
		}
	}
}

// evictOldestEntry removes the oldest entry from cache
func (s *Service) evictOldestEntry() {
	var oldestIP string
	var oldestTime time.Time
	
	first := true
	for ip, info := range s.cache {
		if first || info.CacheTime.Before(oldestTime) {
			oldestIP = ip
			oldestTime = info.CacheTime
			first = false
		}
	}
	
	if oldestIP != "" {
		delete(s.cache, oldestIP)
	}
}

