package regles

import "time"

const day = 24 * time.Hour

type SeniorityPolicy struct {
	MinimumDays int
}

func (p SeniorityPolicy) isNewcomer(seller Seller, submittedAt time.Time) bool {
	return seniorityDays(seller.MemberSince, submittedAt) < p.MinimumDays
}

func seniorityDays(memberSince, submittedAt time.Time) int {
	submissionDay := submittedAt.UTC().Truncate(day)
	return int(submissionDay.Sub(memberSince.UTC().Truncate(day)) / day)
}
