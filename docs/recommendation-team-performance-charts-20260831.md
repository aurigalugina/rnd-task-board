# Rekomendasi Team Performance Monitoring Charts

**Date:** 2026-08-31  
**Context:** Enhance dashboard dengan chart monitoring kinerja tim

---

## 📊 Saat Ini vs. Yang Bisa Ditambah

### Existing Charts
1. **Status Project (Donut)** — Distribusi status (On Progress, Belum, Selesai, Hold)
2. **Hasil Project (Donut)** — Won vs Lose outcome
3. **Progress: Actual vs Expected (Bar)** — Tracking progress per board

---

## 🎯 Rekomendasi Chart Baru (8 options)

### Tier 1: HIGH IMPACT (Critical untuk monitor kinerja tim)

#### **1. Team Velocity / Sprint Trend** ⭐⭐⭐
**Apa:** Line chart menampilkan completion rate trend per minggu/bulan  
**Metric:** `(completed_tasks / total_tasks) * 100` trend over time  
**Benefit:**
- Lihat apakah tim accelerating atau slowing down
- Early warning jika velocity drop
- Regresi vs. Progress tracking

**Visualisasi:**
```
100% ┤     ╱╲    ╭─╮
  75% ┤   ╱    ╲╭─╯   
  50% ┤ ╱╲    ╱
  25% ┤        
   0% ┼─────────────── Week 1,2,3,4
```

**Implementation:** SvelteLine chart (smooth curve, fill under)

---

#### **2. Team Member Performance Matrix** ⭐⭐⭐
**Apa:** 2D scatter plot: (Tasks Completed) vs (Avg Completion Time)  
**Metric:** Each dot = 1 team member  
**Benefit:**
- Quick identify top performers vs. bottlenecks
- Quadrant analysis: Fast & High-Output vs. Slow & Low-Output
- Resource allocation insights

**Visualisasi:**
```
Completion Time (hours)
500 ┤ 🟡(slow,few) . . . 🔴(slow,many)
    ┤
250 ┤           🟢(fast,many)
    ┤ 🟠(fast,few)
  0 ┼────────────────────────────
    0        50      100      150
         Tasks Completed
```

**Implementation:** Bubble chart + quadrant grid

---

#### **3. Blockers / Risk Heatmap** ⭐⭐⭐
**Apa:** Table/Calendar heatmap menampilkan tasks yang "hold" atau overdue  
**Metric:** Count of blocked/overdue tasks per day/week  
**Benefit:**
- Visual overview kapan tim paling sering "stuck"
- Pattern detection (Mondays? End of sprint?)
- Risk mitigation timing

**Visualisasi:**
```
       Mon  Tue  Wed  Thu  Fri
Week1  🟨   🔴   🔴   🟨   🟩
Week2  🟩   🟩   🟨   🟨   🟩
Week3  🔴   🔴   🔴   🟨   🟩
```

**Implementation:** Heatmap grid + color scale (green=good, red=risky)

---

### Tier 2: MEDIUM IMPACT (Nice-to-have, actionable insights)

#### **4. Task Cycle Time Distribution** ⭐⭐
**Apa:** Histogram: "How long does a typical task take?"  
**Metric:** Bucket tasks by completion time (1-3 days, 3-7 days, 1-2 weeks, 2+ weeks)  
**Benefit:**
- Understand task complexity distribution
- Estimate future sprint capacity
- Identify outliers (tasks stuck too long)

**Implementation:** Horizontal bar chart + tooltip for details

---

#### **5. Dependency / Cross-Team Impact Network**
**Apa:** Node-link diagram showing task dependencies between boards  
**Benefit:**
- See which boards block which
- Critical path identification
- Cross-team coordination needs

**Implementation:** Force-directed graph (if data available)

---

#### **6. Win-Loss Trend & Root Cause Breakdown** ⭐⭐
**Apa:** Stacked bar: Weekly Win/Lose breakdown + color-coded reason  
**Metric:** Reason tags (schedule mismatch, scope creep, resource, external blocker)  
**Benefit:**
- Learn WHY projects fail
- Pattern detection in failure modes
- Process improvement roadmap

**Implementation:** Stacked bar + legend with reason colors

---

### Tier 3: ADVANCED / STRATEGIC (Long-term planning)

#### **7. Capacity Planning Forecast** ⭐
**Apa:** Stacked area chart: Available capacity vs. Committed capacity over time  
**Benefit:**
- See if team is over/under-committed
- Capacity runway (can we take on more?)

**Implementation:** Area chart + reference lines

---

#### **8. Team Engagement / Burnout Indicator** ⭐
**Apa:** Multi-metric score: (Task completion rate, Hours worked, Task variety)  
**Metric:** Simple 0-100 score with breakdown  
**Benefit:**
- Monitor team health
- Early warning burnout
- Celebrate wins

**Implementation:** Gauge chart + detail breakdown

---

## 🎯 RECOMMENDED IMPLEMENTATION PLAN

### Phase 1 (Next Sprint) — Core Monitoring
1. **Team Velocity** (Line chart) ← HIGH ROI
2. **Team Performance Matrix** (Scatter plot) ← Quick insights
3. **Blockers Heatmap** (Calendar) ← Risk management

**Why:** These 3 give 80% of actionable insights, relatively easy to implement.

### Phase 2 (Following Sprint) — Deep Dive
4. Task Cycle Time Distribution
5. Win-Loss Trend Breakdown

### Phase 3 (Later) — Strategic
6. Capacity Planning Forecast
7. Team Engagement Indicator

---

## 🔧 Technical Implementation Notes

### Data Collection Needed
- **Velocity:** Timestamp each task completion
- **Performance Matrix:** Calculate avg time per team member
- **Heatmap:** Query blocked/overdue tasks per date
- **Cycle Time:** End date - Start date per task
- **Win-Loss:** Add `failure_reason` field to Big Task

### UI/UX Considerations
- **Mobile responsive:** Charts stack vertically on mobile
- **Interactive legends:** Click to filter/drill-down
- **Hover tooltips:** Show exact numbers
- **Color accessibility:** Use colorblind-friendly palette
- **Real-time updates:** Stream data changes (optional)

### Suggested Chart Library
- **Lightweight:** Use custom SVG + Svelte (current approach)
- **Or:** Recharts (if lightweight needed)
- **Heavy option:** D3.js (if complex interactions needed)

---

## 📈 Expected Outcomes

After implementing these charts:
1. ✅ **Visibility:** Know exactly how team is performing
2. ✅ **Early warning:** Catch issues before they escalate
3. ✅ **Data-driven decisions:** Plan sprints based on actual velocity
4. ✅ **Culture:** Celebrate wins, identify support needs
5. ✅ **Process improvement:** Learn from patterns

---

## 💡 Sample Implementation (Phase 1 - Velocity Chart)

```svelte
<script>
  // Team velocity tracking
  // data = [{ week: '1', completed: 45, total: 80 }, ...]
  export let velocityData;
</script>

<svg viewBox="0 0 640 200" width="100%" height="200" class="velocity-chart">
  <!-- Grid lines, axis -->
  <!-- Path for line -->
  {#each velocityData as d, i}
    <line x1={...} y1={...} x2={...} y2={...} class="velocity-line" />
  {/each}
  <!-- Dots with hover -->
</svg>

<style>
  .velocity-line {
    animation: drawLine 1.2s ease-out;
    stroke: var(--win-blue);
    stroke-width: 2;
    fill: none;
  }
</style>
```

---

**Next Step:** Confirm top 3 priorities with team, then implement! 🚀
