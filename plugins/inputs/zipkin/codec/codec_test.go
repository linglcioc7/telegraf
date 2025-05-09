package codec

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/influxdata/telegraf/plugins/inputs/zipkin/trace"
)

func Test_MicroToTime(t *testing.T) {
	tests := []struct {
		name  string
		micro int64
		want  time.Time
	}{
		{
			name:  "given zero micro seconds expected unix time zero",
			micro: 0,
			want:  time.Unix(0, 0).UTC(),
		},
		{
			name:  "given a million micro seconds expected unix time one",
			micro: 1000000,
			want:  time.Unix(1, 0).UTC(),
		},
		{
			name:  "given a million micro seconds expected unix time one",
			micro: 1503031538791000,
			want:  time.Unix(0, 1503031538791000000).UTC(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, MicroToTime(tt.micro))
		})
	}
}

func Test_minMax(t *testing.T) {
	tests := []struct {
		name    string
		span    *MockSpan
		now     func() time.Time
		wantMin time.Time
		wantMax time.Time
	}{
		{
			name: "Single annotation",
			span: &MockSpan{
				Anno: []Annotation{
					&MockAnnotation{
						Time: time.Unix(0, 0).UTC().Add(time.Second),
					},
				},
			},
			wantMin: time.Unix(1, 0).UTC(),
			wantMax: time.Unix(1, 0).UTC(),
		},
		{
			name: "Three annotations",
			span: &MockSpan{
				Anno: []Annotation{
					&MockAnnotation{
						Time: time.Unix(0, 0).UTC().Add(1 * time.Second),
					},
					&MockAnnotation{
						Time: time.Unix(0, 0).UTC().Add(2 * time.Second),
					},
					&MockAnnotation{
						Time: time.Unix(0, 0).UTC().Add(3 * time.Second),
					},
				},
			},
			wantMin: time.Unix(1, 0).UTC(),
			wantMax: time.Unix(3, 0).UTC(),
		},
		{
			name: "Annotations are in the future",
			span: &MockSpan{
				Anno: []Annotation{
					&MockAnnotation{
						Time: time.Unix(0, 0).UTC().Add(3 * time.Second),
					},
				},
			},
			wantMin: time.Unix(2, 0).UTC(),
			wantMax: time.Unix(3, 0).UTC(),
			now: func() time.Time {
				return time.Unix(2, 0).UTC()
			},
		},
		{
			name:    "No Annotations",
			span:    &MockSpan{},
			wantMin: time.Unix(2, 0).UTC(),
			wantMax: time.Unix(2, 0).UTC(),
			now: func() time.Time {
				return time.Unix(2, 0).UTC()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.now != nil {
				now = tt.now
			}
			gotMin, gotMax := minMax(tt.span)
			require.Equal(t, tt.wantMin, gotMin)
			require.Equal(t, tt.wantMax, gotMax)

			now = time.Now
		})
	}
}

func Test_guessTimestamp(t *testing.T) {
	tests := []struct {
		name string
		span Span
		now  func() time.Time
		want time.Time
	}{
		{
			name: "simple timestamp",
			span: &MockSpan{
				Time: time.Unix(2, 0).UTC(),
			},
			want: time.Unix(2, 0).UTC(),
		},
		{
			name: "zero timestamp",
			span: &MockSpan{
				Time: time.Time{},
			},
			now: func() time.Time {
				return time.Unix(2, 0).UTC()
			},
			want: time.Unix(2, 0).UTC(),
		},
		{
			name: "zero timestamp with single annotation",
			span: &MockSpan{
				Time: time.Time{},
				Anno: []Annotation{
					&MockAnnotation{
						Time: time.Unix(0, 0).UTC(),
					},
				},
			},
			want: time.Unix(0, 0).UTC(),
		},
		{
			name: "zero timestamp with two annotations",
			span: &MockSpan{
				Time: time.Time{},
				Anno: []Annotation{
					&MockAnnotation{
						Time: time.Unix(0, 0).UTC(),
					},
					&MockAnnotation{
						Time: time.Unix(2, 0).UTC(),
					},
				},
			},
			want: time.Unix(0, 0).UTC(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.now != nil {
				now = tt.now
			}
			require.Equal(t, tt.want, guessTimestamp(tt.span))
			now = time.Now
		})
	}
}

func Test_convertDuration(t *testing.T) {
	tests := []struct {
		name string
		span Span
		want time.Duration
	}{
		{
			name: "simple duration",
			span: &MockSpan{
				Dur: time.Hour,
			},
			want: time.Hour,
		},
		{
			name: "no timestamp, but, 2 seconds between annotations",
			span: &MockSpan{
				Anno: []Annotation{
					&MockAnnotation{
						Time: time.Unix(0, 0).UTC().Add(1 * time.Second),
					},
					&MockAnnotation{
						Time: time.Unix(0, 0).UTC().Add(2 * time.Second),
					},
					&MockAnnotation{
						Time: time.Unix(0, 0).UTC().Add(3 * time.Second),
					},
				},
			},
			want: 2 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, convertDuration(tt.span))
		})
	}
}

func Test_parentID(t *testing.T) {
	tests := []struct {
		name    string
		span    Span
		want    string
		wantErr bool
	}{
		{
			name: "has parent id",
			span: &MockSpan{
				ParentID: "6b221d5bc9e6496c",
			},
			want: "6b221d5bc9e6496c",
		},
		{
			name: "no parent, so use id",
			span: &MockSpan{
				ID: "abceasyas123",
			},
			want: "abceasyas123",
		},
		{
			name: "bad parent value",
			span: &MockSpan{
				Error: errors.New("mommie dearest"),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parentID(tt.span)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_serviceEndpoint(t *testing.T) {
	tests := []struct {
		name string
		ann  []Annotation
		bann []BinaryAnnotation
		want Endpoint
	}{
		{
			name: "Annotation with server receive",
			ann: []Annotation{
				&MockAnnotation{
					Val: "battery",
					H: &MockEndpoint{
						name: "aa",
					},
				},
				&MockAnnotation{
					Val: "sr",
					H: &MockEndpoint{
						name: "me",
					},
				},
			},
			want: &MockEndpoint{
				name: "me",
			},
		},
		{
			name: "Annotation with no standard values",
			ann: []Annotation{
				&MockAnnotation{
					Val: "noop",
				},
				&MockAnnotation{
					Val: "aa",
					H: &MockEndpoint{
						name: "battery",
					},
				},
			},
			want: &MockEndpoint{
				name: "battery",
			},
		},
		{
			name: "Annotation with no endpoints",
			ann: []Annotation{
				&MockAnnotation{
					Val: "noop",
				},
			},
			want: &defaultEndpoint{},
		},
		{
			name: "Binary annotation with local component",
			bann: []BinaryAnnotation{
				&MockBinaryAnnotation{
					K: "noop",
					H: &MockEndpoint{
						name: "aa",
					},
				},
				&MockBinaryAnnotation{
					K: "lc",
					H: &MockEndpoint{
						name: "me",
					},
				},
			},
			want: &MockEndpoint{
				name: "me",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, serviceEndpoint(tt.ann, tt.bann))
		})
	}
}

func TestNewBinaryAnnotations(t *testing.T) {
	tests := []struct {
		name        string
		annotations []BinaryAnnotation
		endpoint    Endpoint
		want        []trace.BinaryAnnotation
	}{
		{
			name: "Should override annotation with endpoint",
			annotations: []BinaryAnnotation{
				&MockBinaryAnnotation{
					K: "mykey",
					V: "myvalue",
					H: &MockEndpoint{
						host: "noop",
						name: "noop",
					},
				},
			},
			endpoint: &MockEndpoint{
				host: "myhost",
				name: "myservice",
			},
			want: []trace.BinaryAnnotation{
				{
					Host:        "myhost",
					ServiceName: "myservice",
					Key:         "mykey",
					Value:       "myvalue",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, NewBinaryAnnotations(tt.annotations, tt.endpoint))
		})
	}
}

func TestNewAnnotations(t *testing.T) {
	tests := []struct {
		name        string
		annotations []Annotation
		endpoint    Endpoint
		want        []trace.Annotation
	}{
		{
			name: "Should override annotation with endpoint",
			annotations: []Annotation{
				&MockAnnotation{
					Time: time.Unix(0, 0).UTC(),
					Val:  "myvalue",
					H: &MockEndpoint{
						host: "noop",
						name: "noop",
					},
				},
			},
			endpoint: &MockEndpoint{
				host: "myhost",
				name: "myservice",
			},
			want: []trace.Annotation{
				{
					Host:        "myhost",
					ServiceName: "myservice",
					Timestamp:   time.Unix(0, 0).UTC(),
					Value:       "myvalue",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, NewAnnotations(tt.annotations, tt.endpoint))
		})
	}
}

func TestNewTrace(t *testing.T) {
	tests := []struct {
		name    string
		spans   []Span
		now     func() time.Time
		want    trace.Trace
		wantErr bool
	}{
		{
			name: "empty span",
			spans: []Span{
				&MockSpan{},
			},
			now: func() time.Time {
				return time.Unix(0, 0).UTC()
			},
			want: trace.Trace{
				trace.Span{
					ServiceName:       "unknown",
					Timestamp:         time.Unix(0, 0).UTC(),
					Annotations:       make([]trace.Annotation, 0),
					BinaryAnnotations: make([]trace.BinaryAnnotation, 0),
				},
			},
		},
		{
			name: "span has no id",
			spans: []Span{
				&MockSpan{
					Error: errors.New("span has no id"),
				},
			},
			wantErr: true,
		},
		{
			name: "complete span",
			spans: []Span{
				&MockSpan{
					TraceID:     "tid",
					ID:          "id",
					ParentID:    "",
					ServiceName: "me",
					Anno: []Annotation{
						&MockAnnotation{
							Time: time.Unix(1, 0).UTC(),
							Val:  "myval",
							H: &MockEndpoint{
								host: "myhost",
								name: "myname",
							},
						},
					},
					Time: time.Unix(0, 0).UTC(),
					Dur:  2 * time.Second,
				},
			},
			now: func() time.Time {
				return time.Unix(0, 0).UTC()
			},
			want: trace.Trace{
				trace.Span{
					ID:          "id",
					ParentID:    "id",
					TraceID:     "tid",
					Name:        "me",
					ServiceName: "myname",
					Timestamp:   time.Unix(0, 0).UTC(),
					Duration:    2 * time.Second,
					Annotations: []trace.Annotation{
						{
							Timestamp:   time.Unix(1, 0).UTC(),
							Value:       "myval",
							Host:        "myhost",
							ServiceName: "myname",
						},
					},
					BinaryAnnotations: make([]trace.BinaryAnnotation, 0),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.now != nil {
				now = tt.now
			}
			got, err := NewTrace(tt.spans)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.want, got)
			now = time.Now
		})
	}
}

type MockSpan struct {
	TraceID     string
	ID          string
	ParentID    string
	ServiceName string
	Anno        []Annotation
	BinAnno     []BinaryAnnotation
	Time        time.Time
	Dur         time.Duration
	Error       error
}

func (m *MockSpan) Trace() (string, error) {
	return m.TraceID, m.Error
}

func (m *MockSpan) SpanID() (string, error) {
	return m.ID, m.Error
}

func (m *MockSpan) Parent() (string, error) {
	return m.ParentID, m.Error
}

func (m *MockSpan) Name() string {
	return m.ServiceName
}

func (m *MockSpan) Annotations() []Annotation {
	return m.Anno
}

func (m *MockSpan) BinaryAnnotations() ([]BinaryAnnotation, error) {
	return m.BinAnno, m.Error
}

func (m *MockSpan) Timestamp() time.Time {
	return m.Time
}

func (m *MockSpan) Duration() time.Duration {
	return m.Dur
}

type MockAnnotation struct {
	Time time.Time
	Val  string
	H    Endpoint
}

func (m *MockAnnotation) Timestamp() time.Time {
	return m.Time
}

func (m *MockAnnotation) Value() string {
	return m.Val
}

func (m *MockAnnotation) Host() Endpoint {
	return m.H
}

type MockEndpoint struct {
	host string
	name string
}

func (e *MockEndpoint) Host() string {
	return e.host
}

func (e *MockEndpoint) Name() string {
	return e.name
}

type MockBinaryAnnotation struct {
	Time time.Time
	K    string
	V    string
	H    Endpoint
}

func (b *MockBinaryAnnotation) Key() string {
	return b.K
}

func (b *MockBinaryAnnotation) Value() string {
	return b.V
}

func (b *MockBinaryAnnotation) Host() Endpoint {
	return b.H
}
