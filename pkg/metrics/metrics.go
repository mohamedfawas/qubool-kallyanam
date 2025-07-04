package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all business metrics for the application
type Metrics struct {
	// User lifecycle metrics
	UserRegistrations prometheus.Counter
	UserVerifications prometheus.Counter
	UserLogins        prometheus.Counter

	// Matchmaking metrics
	MatchesLiked  prometheus.Counter
	MatchesPassed prometheus.Counter
	MutualMatches prometheus.Counter

	// Revenue metrics
	PaymentOrdersCreated    prometheus.Counter
	PaymentOrdersCompleted  prometheus.Counter
	SubscriptionActivations prometheus.Counter

	// Engagement metrics
	ConversationsCreated prometheus.Counter

	// Basic health metrics
	HTTPRequests *prometheus.CounterVec
}

// New creates a new metrics registry for the given service
func New(serviceName string) *Metrics {
	return &Metrics{
		// User metrics
		UserRegistrations: promauto.NewCounter(prometheus.CounterOpts{
			Name:        "user_registrations_total",
			Help:        "Total number of user registrations",
			ConstLabels: prometheus.Labels{"service": serviceName},
		}),

		UserVerifications: promauto.NewCounter(prometheus.CounterOpts{
			Name:        "user_verifications_total",
			Help:        "Total number of user email verifications",
			ConstLabels: prometheus.Labels{"service": serviceName},
		}),

		UserLogins: promauto.NewCounter(prometheus.CounterOpts{
			Name:        "user_logins_total",
			Help:        "Total number of successful user logins",
			ConstLabels: prometheus.Labels{"service": serviceName},
		}),

		// Matchmaking metrics
		MatchesLiked: promauto.NewCounter(prometheus.CounterOpts{
			Name:        "matches_liked_total",
			Help:        "Total number of profiles liked by users",
			ConstLabels: prometheus.Labels{"service": serviceName},
		}),

		MatchesPassed: promauto.NewCounter(prometheus.CounterOpts{
			Name:        "matches_passed_total",
			Help:        "Total number of profiles passed by users",
			ConstLabels: prometheus.Labels{"service": serviceName},
		}),

		MutualMatches: promauto.NewCounter(prometheus.CounterOpts{
			Name:        "mutual_matches_total",
			Help:        "Total number of mutual matches between users",
			ConstLabels: prometheus.Labels{"service": serviceName},
		}),

		// Payment metrics
		PaymentOrdersCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name:        "payment_orders_created_total",
			Help:        "Total number of payment orders created",
			ConstLabels: prometheus.Labels{"service": serviceName},
		}),

		PaymentOrdersCompleted: promauto.NewCounter(prometheus.CounterOpts{
			Name:        "payment_orders_completed_total",
			Help:        "Total number of payment orders completed successfully",
			ConstLabels: prometheus.Labels{"service": serviceName},
		}),

		SubscriptionActivations: promauto.NewCounter(prometheus.CounterOpts{
			Name:        "subscription_activations_total",
			Help:        "Total number of premium subscription activations",
			ConstLabels: prometheus.Labels{"service": serviceName},
		}),

		// Engagement metrics
		ConversationsCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name:        "conversations_created_total",
			Help:        "Total number of chat conversations created",
			ConstLabels: prometheus.Labels{"service": serviceName},
		}),

		// Basic HTTP health
		HTTPRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "http_requests_total",
				Help:        "Total number of HTTP requests",
				ConstLabels: prometheus.Labels{"service": serviceName},
			},
			[]string{"status"},
		),
	}
}

// Business metric helper methods
func (m *Metrics) IncrementUserRegistrations() {
	m.UserRegistrations.Inc()
}

func (m *Metrics) IncrementUserVerifications() {
	m.UserVerifications.Inc()
}

func (m *Metrics) IncrementUserLogins() {
	m.UserLogins.Inc()
}

func (m *Metrics) IncrementMatchesLiked() {
	m.MatchesLiked.Inc()
}

func (m *Metrics) IncrementMatchesPassed() {
	m.MatchesPassed.Inc()
}

func (m *Metrics) IncrementMutualMatches() {
	m.MutualMatches.Inc()
}

func (m *Metrics) IncrementPaymentOrdersCreated() {
	m.PaymentOrdersCreated.Inc()
}

func (m *Metrics) IncrementPaymentOrdersCompleted() {
	m.PaymentOrdersCompleted.Inc()
}

func (m *Metrics) IncrementSubscriptionActivations() {
	m.SubscriptionActivations.Inc()
}

func (m *Metrics) IncrementConversationsCreated() {
	m.ConversationsCreated.Inc()
}
