package metrics_test

import (
	"math"
	"testing"
	"time"

	"github.com/socialpoint-labs/bsk/metrics"
	"github.com/stretchr/testify/assert"
)

func TestMetricsRecorderRegistry(t *testing.T) {
	a := assert.New(t)

	r := metrics.NewRecorder()

	c := r.Counter("counter")
	c.Inc()
	a.Equal(c, r.Get("counter")[0])

	g := r.Gauge("gauge")
	a.Equal(g, r.Get("gauge")[0])

	timer := r.Timer("timer")
	a.Equal(timer, r.Get("timer")[0])

	a.Nil(r.Get("does-not-exists"))
}

func TestMetricsRecorder(t *testing.T) {
	a := assert.New(t)

	moreTags := metrics.Tags{
		metrics.NewTag("moretag1", "value1"),
		metrics.NewTag("moretag2", "value2"),
	}

	lastTagKey := "lastTagKey"
	lastTagValue := "lastTagValue"
	lastTag := metrics.NewTag(lastTagKey, lastTagValue)

	for _, tags := range []metrics.Tags{
		{},
		{metrics.Tag{Key: "foo", Value: "bar"}},
		{metrics.Tag{Key: "foo", Value: "bar"}, metrics.Tag{Key: "foo2", Value: "bar2"}},
		{metrics.NewTag("foo", "bar"), metrics.NewTag("foo2", "bar2")},
		{metrics.NewTag("foo", "bar"), metrics.NewTag("foo2", "bar2")},
	} {
		r := metrics.NewRecorder()

		// test counter inc
		metricName := "counter"
		c := r.Counter(metricName, tags...).(*metrics.RecorderCounter)
		c.Inc()
		c.Inc()

		a.EqualValues(2, c.Value)

		// test counter tags
		a.Equal(c.Tags(), tags)
		c.WithTags(moreTags...) // another way to set tags

		c.Inc()
		a.EqualValues(3, c.Value)

		a.Equal(append(tags, moreTags...), c.Tags())

		// test counter add from inc
		c.Add(10)
		a.EqualValues(13, c.Value)

		// test counter add

		c = r.Counter(metricName, tags...).(*metrics.RecorderCounter)
		c.WithTags(tags...)
		c.Add(10)
		a.EqualValues(23, c.Value)

		// test gauge
		metricName = "gauge"
		g := r.Gauge(metricName, tags...).(*metrics.RecorderGauge)
		g.Update(math.Pi)
		a.Equal(math.Pi, g.Value)
		g.Update(math.E)
		a.EqualValues(math.E, g.Value)

		// test gauge tags
		a.Equal(g.Tags(), tags)
		g.WithTags(moreTags...) // another way to set tags
		g.Update(math.Ln2)
		a.EqualValues(math.Ln2, g.Value)
		a.EqualValues(g.Tags(), append(tags, moreTags...))
		g.WithTag(lastTagKey, lastTagValue) // and another way to add one tag
		a.Equal(g.Tags(), append(append(tags, moreTags...), lastTag))

		// test event
		metricName = "event"
		e := r.Event(metricName, tags...).(*metrics.RecorderEvent)
		e.Send()
		a.Equal("event|", e.Event)
		e.SendWithText("msg")
		a.Equal("event|msg", e.Event)

		// test event tags
		a.Equal(e.Tags(), tags)
		e.WithTags(moreTags...) // another way to set tags
		e.SendWithText("msg2")
		a.Equal("event|msg2", e.Event)
		a.Equal(append(tags, moreTags...), e.Tags())
		e.WithTag(lastTagKey, lastTagValue) // and another way to add one tag
		a.Equal(e.Tags(), append(append(tags, moreTags...), lastTag))

		// test Timer
		metricName = "timer"
		t := r.Timer(metricName, tags...)
		t.Start()
		t.Stop()

		// test timer tags
		a.Equal(t.Tags(), tags)
		t.WithTags(moreTags...) // another way to set tags
		a.Equal(t.Tags(), append(tags, moreTags...))
		t.WithTag(lastTagKey, lastTagValue) // and another way to add one tag
		a.Equal(t.Tags(), append(append(tags, moreTags...), lastTag))

		// test Histogram
		metricName = "histogram"
		h := r.Histogram(metricName, tags...).(*metrics.RecorderHistogram)
		h.AddValue(42)
		h.AddValue(666)
		a.Equal([]uint64{42, 666}, h.Values)

		// test histogram tags
		a.Equal(h.Tags(), tags)
		h.WithTags(moreTags...) // another way to set tags
		a.Equal(h.Tags(), append(tags, moreTags...))
		h.WithTag(lastTagKey, lastTagValue) // and another way to add one tag
		a.Equal(h.Tags(), append(append(tags, moreTags...), lastTag))
	}
}

func TestMetricsRecorder_tags(t *testing.T) {
	a := assert.New(t)
	r := metrics.NewRecorder()

	tag1 := metrics.NewTag("foo", "bar1")
	tag2 := metrics.NewTag("foo", "bar2")

	// test counter inc
	metricName := "counter"
	c1 := r.Counter(metricName)
	c1.WithTags(tag1)
	c1.Inc()
	r.Counter(metricName, tag2).Inc()
	r.Counter(metricName, tag1).Inc()

	counter := r.GetWithTags(metricName, tag1).(*metrics.RecorderCounter)
	a.Equal(uint64(2), counter.Value)
	a.Equal(metrics.Tags([]metrics.Tag{tag1}), counter.Tags())

	counter = r.GetWithTags(metricName, tag2).(*metrics.RecorderCounter)
	a.Equal(uint64(1), counter.Value)
	a.Equal(metrics.Tags([]metrics.Tag{tag2}), counter.Tags())
}

func TestRecorder_ConcurrentSafety(t *testing.T) {
	a := assert.New(t)
	r := metrics.NewRecorder()

	ch := make(chan bool)

	// Register several types of metrics
	r.Counter("counter").Inc()
	r.Gauge("gauge")
	r.Timer("timer")
	r.Event("event")
	r.Histogram("histogram")

	thread := func() {
		c := r.Get("counter")[0].(*metrics.RecorderCounter)
		c.Inc()

		g := r.Get("gauge")[0].(*metrics.RecorderGauge)
		g.Update(123)

		timer := r.Get("timer")[0].(*metrics.RecorderTimer)
		timer.Start()
		timer.Stop()

		e := r.Get("event")[0].(*metrics.RecorderEvent)
		e.SendWithText("life")

		h := r.Get("histogram")[0].(*metrics.RecorderHistogram)
		h.AddValue(42)
		h.AddValue(666)

		ch <- true
	}

	c := r.Get("counter")[0].(*metrics.RecorderCounter)
	g := r.Get("gauge")[0].(*metrics.RecorderGauge)
	timer := r.Get("timer")[0].(*metrics.RecorderTimer)
	h := r.Get("histogram")[0].(*metrics.RecorderHistogram)

	go thread()
	go thread()

	<-ch
	<-ch

	a.EqualValues(3, c.Value)
	a.EqualValues(123, g.Value)
	a.WithinDuration(timer.StartedTime, timer.StoppedTime, time.Duration(time.Millisecond))
	a.Equal([]uint64{42, 666, 42, 666}, h.Values)
}
