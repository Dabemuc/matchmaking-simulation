This is a **very strong metrics setup** already. You’re clearly thinking like a systems engineer, not just “add some counters”. I’ll break this down into:

1. **Overall assessment**
2. **What’s missing / what I’d tweak**
3. **Subtle correctness issues**
4. **Dashboards I would build (very concrete)**
5. **One optional “next-level” improvement**

---

## 1️⃣ Overall assessment

Short version: **this is production-grade for a simulation harness**.

You have:

- Correct metric _types_ (gauges vs counters vs histograms)
- Bounded cardinality
- Metrics aligned with actual control points (pool, compositor, scenarios)
- Backpressure visibility
- Cancellation & shutdown observability
- Self-observability (goroutines, tick durations)

This is **better than what many real services ship**.

---

## 2️⃣ What’s missing / what I would add

### 🔹 A) Scenario _attempted_ vs _executed_

Right now you have:

- `scenario_started_total` → when a player actually runs it
- `compositor_idle_starvation_total` → when scheduling fails

What’s missing is an explicit **“attempted executions”** metric.

#### Why it matters

You want to see:

```
attempted > started → starvation
attempted == started → healthy
```

### Suggested metric

```go
scenarioAttemptedTotal = promauto.NewCounterVec(
    prometheus.CounterOpts{
        Namespace: "harness",
        Name:      "scenario_attempted_total",
        Help:      "Scenario executions requested by compositor.",
    },
    []string{"scenario"},
)
```

Increment this **before** calling `ExecuteScenario`.

---

### 🔹 B) Pool capacity metric

You expose:

- `pool_idle_queue_depth`

But not:

- **maximum idle capacity**

Without it, dashboards can’t show saturation percentage.

### Suggested metric

```go
poolIdleCapacity = promauto.NewGauge(
    prometheus.GaugeOpts{
        Namespace: "harness",
        Name:      "pool_idle_capacity",
        Help:      "Maximum idle pool capacity.",
    },
)
```

Set once in `New()`.

---

### 🔹 C) Player creation rate

You have total and active, but not **rate**.

PromQL can derive it, but it’s a _very important_ signal.

No new metric needed, but you should plan to graph:

```promql
rate(harness_players_total[5s])
```

I’d explicitly plan a dashboard panel for this.

---

### 🔹 D) Scenario cancellation vs failure distinction

Currently:

```go
if err != nil {
    scenarioCompletedTotal.WithLabelValues(s.Name(), "failure").Inc()
}
```

But `ctx.Err()` during shutdown is not really a “failure”.

### Suggested statuses

- `success`
- `failure`
- `cancelled`

This makes shutdown behavior _much_ clearer.

---

### 🔹 E) Compositor tick drift

You measure tick duration (excellent), but not **tick skew**.

If ticks are supposed to be 1s and start drifting, you won’t see it.

Optional metric:

```go
compositorTickLagSeconds
```

But this is optional unless you expect extreme load.

---

## 3️⃣ Subtle correctness issues (important)

These aren’t conceptual problems — just things to tighten.

---

### ⚠️ `playersIdle` gauge correctness

You do:

```go
playersIdle.Inc()   // when putting into idle
playersIdle.Dec()   // when taking from idle
```

This is correct **only if**:

- Every `idle <- p` is matched with exactly one take
- No player exits while idle without decrement

But in this block:

```go
case idle <- p:
    playersIdle.Inc()
case <-ctx.Done():
    return
}
```

If context cancels **after** player is idle but **before** pool pulls it, you leak the gauge.

#### Safer pattern

Track idle count from `len(p.idle)` in the monitor loop only and **remove manual Inc/Dec**.

You already have:

```go
poolIdleQueueDepth.Set(float64(len(p.idle)))
```

I would **delete `playersIdle` entirely** and rely on that.

---

### ⚠️ `playerCnt` is not atomic

`PlayerCount()` returns `p.playerCnt`, but it’s written concurrently.

Prometheus reads don’t care, but correctness does.

Fix:

- Either make it atomic
- Or derive active players from `playersActive` gauge

I’d remove `PlayerCount()` entirely and base compositor logic on `playersActive`.

---

## 4️⃣ Dashboards I would build

Here’s the **exact dashboard structure** I’d ship.

---

## 📊 Dashboard 1: “System Overview”

**Row: Load**

- Active Players → `harness_players_active`
- Player Creation Rate → `rate(harness_players_total[5s])`

**Row: Pool Health**

- Idle Pool Depth → `harness_pool_idle_queue_depth`
- Idle Saturation →

```promql
harness_pool_idle_queue_depth / harness_pool_idle_capacity
```

**Row: Errors**

- Internal Errors →

```promql
rate(harness_errors_total[5m])
```

---

## 📊 Dashboard 2: “Scenario Throughput”

For each scenario (templated variable):

**Panels**

- Desired Rate → `harness_compositor_desired_rate`
- Attempted vs Started →

```promql
rate(harness_scenario_attempted_total[5s])
rate(harness_scenario_started_total[5s])
```

- Completions →

```promql
rate(harness_scenario_completed_total{status="success"}[5s])
```

This tells you instantly if scheduling is working.

---

## 📊 Dashboard 3: “Scenario Latency”

Per scenario:

- P50 / P95 / P99 →

```promql
histogram_quantile(
  0.95,
  rate(harness_scenario_duration_seconds_bucket[5m])
)
```

Overlay all scenarios or split by panel.

---

## 📊 Dashboard 4: “Backpressure & Saturation”

This is the _money dashboard_.

- Idle Starvation Rate →

```promql
rate(harness_compositor_idle_starvation_total[5s])
```

- Pool Execute Wait P95 →

```promql
histogram_quantile(
  0.95,
  rate(harness_pool_execute_wait_duration_seconds_bucket[5m])
)
```

If these go up → system overloaded.

---

## 📊 Dashboard 5: “Scheduler Health”

- Tick Duration (pool & compositor)
- Goroutines
- Context Cancellations by source

This answers:

> “Is my harness itself breaking?”
