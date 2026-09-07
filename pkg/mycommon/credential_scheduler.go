package mycommon

import (
	"sort"
	"sync"
	"time"

	"simple-one-api/pkg/config"
)

const credentialCapacityWindow = time.Minute

type credentialReservation struct {
	at   time.Time
	cost int
}

type credentialCapacityState struct {
	reservations  []credentialReservation
	cooldownUntil time.Time
}

// CredentialCapacitySnapshot is a read-only view of one credential's
// rolling TPM window. It intentionally contains no credential secret.
type CredentialCapacitySnapshot struct {
	TPMLimit        float64
	ReservedTokens  int64
	RemainingTokens int64
	Available       bool
	CooldownUntil   *time.Time
	AvailableAt     *time.Time
}

var credentialCapacity = struct {
	sync.Mutex
	states map[string]*credentialCapacityState
}{states: make(map[string]*credentialCapacityState)}

type credentialCandidateScore struct {
	candidate     CredentialSelection
	available     bool
	remaining     float64
	nextAvailable time.Time
	order         int
}

// OrderCredentialCandidates prioritizes keys that can accommodate the request
// within their configured model/key TPM window, then keys with the most
// remaining capacity. It preserves the provider's base ordering for ties.
func OrderCredentialCandidates(service *config.ModelDetails, model string, tokenCost int) []CredentialSelection {
	candidates := GetCredentialCandidates(service, model)
	if len(candidates) < 2 {
		return candidates
	}
	if tokenCost < 1 {
		tokenCost = 1
	}
	scores := make([]credentialCandidateScore, 0, len(candidates))
	for index, candidate := range candidates {
		limit := credentialTPMLimit(candidate.Credentials, model)
		available, remaining, next := credentialCapacityAvailability(candidate.ID, model, limit, tokenCost)
		scores = append(scores, credentialCandidateScore{
			candidate: candidate, available: available, remaining: remaining,
			nextAvailable: next, order: index,
		})
	}
	sort.SliceStable(scores, func(left, right int) bool {
		a, b := scores[left], scores[right]
		if a.available != b.available {
			return a.available
		}
		if a.available && a.remaining != b.remaining {
			return a.remaining > b.remaining
		}
		if !a.available && !a.nextAvailable.Equal(b.nextAvailable) {
			return a.nextAvailable.Before(b.nextAvailable)
		}
		return a.order < b.order
	})
	ordered := make([]CredentialSelection, len(scores))
	for index, score := range scores {
		ordered[index] = score.candidate
	}
	return ordered
}

// ReserveCredentialCapacity records the conservative request estimate after
// the normal limiter has accepted the attempt. Reservations remain in the
// rolling window because upstream TPM accounting is not refundable.
func ReserveCredentialCapacity(credentials map[string]interface{}, id, model string, tokenCost int) {
	limit := credentialTPMLimit(credentials, model)
	if limit <= 0 {
		return
	}
	if tokenCost < 1 {
		tokenCost = 1
	}
	now := time.Now()
	key := id + "\x00" + model
	credentialCapacity.Lock()
	defer credentialCapacity.Unlock()
	state := credentialCapacity.states[key]
	if state == nil {
		state = &credentialCapacityState{}
		credentialCapacity.states[key] = state
	}
	state.reservations = pruneCredentialReservations(state.reservations, now)
	state.reservations = append(state.reservations, credentialReservation{at: now, cost: tokenCost})
}

func MarkCredentialCooldown(id, model string, duration time.Duration) {
	if duration <= 0 {
		duration = 30 * time.Second
	}
	until := time.Now().Add(duration)
	key := id + "\x00" + model
	credentialCapacity.Lock()
	defer credentialCapacity.Unlock()
	state := credentialCapacity.states[key]
	if state == nil {
		state = &credentialCapacityState{}
		credentialCapacity.states[key] = state
	}
	if until.After(state.cooldownUntil) {
		state.cooldownUntil = until
	}
}

// InspectCredentialCapacity returns the current runtime capacity for a
// credential/model pair. The reservation window is pruned while reading so
// the admin UI reflects the same state used by request scheduling.
func InspectCredentialCapacity(credentials map[string]interface{}, id, model string) CredentialCapacitySnapshot {
	limit := credentialTPMLimit(credentials, model)
	now := time.Now()
	key := id + "\x00" + model

	credentialCapacity.Lock()
	defer credentialCapacity.Unlock()

	state := credentialCapacity.states[key]
	if state == nil {
		remaining := int64(0)
		if limit > 0 {
			remaining = int64(limit)
		}
		return CredentialCapacitySnapshot{
			TPMLimit:        limit,
			RemainingTokens: remaining,
			Available:       true,
		}
	}

	state.reservations = pruneCredentialReservations(state.reservations, now)
	used := 0
	nextReservation := time.Time{}
	for _, reservation := range state.reservations {
		used += reservation.cost
		availableAt := reservation.at.Add(credentialCapacityWindow)
		if nextReservation.IsZero() || availableAt.Before(nextReservation) {
			nextReservation = availableAt
		}
	}
	remaining := int64(0)
	if limit > 0 {
		remaining = int64(limit) - int64(used)
		if remaining < 0 {
			remaining = 0
		}
	}

	var cooldownUntil *time.Time
	if state.cooldownUntil.After(now) {
		value := state.cooldownUntil
		cooldownUntil = &value
	}
	available := cooldownUntil == nil && (limit <= 0 || remaining > 0)
	var availableAt *time.Time
	if !available {
		next := nextReservation
		if cooldownUntil != nil && (next.IsZero() || next.Before(*cooldownUntil)) {
			next = *cooldownUntil
		}
		if !next.IsZero() {
			availableAt = &next
		}
	}

	return CredentialCapacitySnapshot{
		TPMLimit:        limit,
		ReservedTokens:  int64(used),
		RemainingTokens: remaining,
		Available:       available,
		CooldownUntil:   cooldownUntil,
		AvailableAt:     availableAt,
	}
}

func credentialTPMLimit(credentials map[string]interface{}, model string) float64 {
	if limit, _, ok := GetCredentialModelLimit(credentials, model); ok && limit.TPM > 0 {
		return limit.TPM
	}
	return GetCredentialLimits(credentials).TPM
}

func credentialCapacityAvailability(id, model string, limit float64, tokenCost int) (bool, float64, time.Time) {
	now := time.Now()
	key := id + "\x00" + model
	credentialCapacity.Lock()
	defer credentialCapacity.Unlock()
	state := credentialCapacity.states[key]
	if state == nil {
		if limit <= 0 {
			return true, 0, time.Time{}
		}
		return tokenCost <= int(limit), limit, time.Time{}
	}
	state.reservations = pruneCredentialReservations(state.reservations, now)
	used := 0
	for _, reservation := range state.reservations {
		used += reservation.cost
	}
	remaining := limit - float64(used)
	if limit <= 0 {
		if state.cooldownUntil.After(now) {
			return false, 0, state.cooldownUntil
		}
		return true, 0, time.Time{}
	}
	if float64(tokenCost) <= remaining && (state.cooldownUntil.IsZero() || !state.cooldownUntil.After(now)) {
		return true, remaining, time.Time{}
	}

	// Find the first reservation expiry at which this request would fit. A
	// large request may need several older reservations to leave the window.
	usedAfterExpiry := used
	var capacityAt time.Time
	for _, reservation := range state.reservations {
		usedAfterExpiry -= reservation.cost
		if float64(tokenCost) <= limit-float64(usedAfterExpiry) {
			capacityAt = reservation.at.Add(credentialCapacityWindow)
			break
		}
	}
	next := capacityAt
	if state.cooldownUntil.After(now) && (next.IsZero() || next.Before(state.cooldownUntil)) {
		next = state.cooldownUntil
	}
	return false, remaining, next
}

func pruneCredentialReservations(reservations []credentialReservation, now time.Time) []credentialReservation {
	cutoff := now.Add(-credentialCapacityWindow)
	first := 0
	for first < len(reservations) && reservations[first].at.Before(cutoff) {
		first++
	}
	if first == 0 {
		return reservations
	}
	return append([]credentialReservation(nil), reservations[first:]...)
}
