import React, { useState, useMemo, useRef, useEffect } from "react";
import {
  LayoutDashboard,
  LayoutGrid,
  ClipboardCheck,
  Plus,
  X,
  Check,
  File,
  MessageSquare,
  Link2,
  StickyNote,
  Upload,
  CalendarClock,
  Bell,
  UserRound,
  Settings,
  HelpCircle,
  LogOut,
} from "lucide-react";
import {
  PieChart,
  Pie,
  Cell,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from "recharts";

const TODAY = new Date("2026-08-08");

const THEMES = {
  "retro-light": {
    label: "Retro light",
    modern: false,
    face: "#DEE3E8",
    contentBg: "#FFFFFF",
    contentAlt: "#EDF1F4",
    textPrimary: "#000000",
    textMuted: "#4D4D4D",
    titlebarA: "#34506E",
    titlebarB: "#8FADC2",
    winBlue: "#34506E",
    winBlueLight: "#DDE7EC",
    winGreen: "#3E7D6E",
    winRed: "#B85C52",
    winAmber: "#9E8A5C",
  },
  "retro-dark": {
    label: "Retro dark",
    modern: false,
    face: "#232B33",
    contentBg: "#1B2229",
    contentAlt: "#28323B",
    textPrimary: "#E7EDF2",
    textMuted: "#9FB0BD",
    titlebarA: "#0F2E44",
    titlebarB: "#3D6688",
    winBlue: "#6FAEDE",
    winBlueLight: "#2B3F52",
    winGreen: "#6FC2A8",
    winRed: "#E29184",
    winAmber: "#D8BE85",
  },
  "modern-light": {
    label: "Modern light",
    modern: true,
    face: "#F3F3F3",
    contentBg: "#FFFFFF",
    contentAlt: "#F7F7F7",
    textPrimary: "#1B1B1B",
    textMuted: "#616161",
    titlebarA: "#FFFFFF",
    titlebarB: "#FFFFFF",
    winBlue: "#005FB8",
    winBlueLight: "#E5F2FC",
    winGreen: "#0F7B0F",
    winRed: "#C42B1C",
    winAmber: "#9D5D00",
  },
  "modern-dark": {
    label: "Modern dark",
    modern: true,
    face: "#202020",
    contentBg: "#2B2B2B",
    contentAlt: "#252525",
    textPrimary: "#F3F3F3",
    textMuted: "#B0B0B0",
    titlebarA: "#202020",
    titlebarB: "#202020",
    winBlue: "#60CDFF",
    winBlueLight: "#233240",
    winGreen: "#6CCB5F",
    winRed: "#FF99A4",
    winAmber: "#FCE100",
  },
};

const TEAM = [
  { id: "lugina", name: "Lugina", role: "SPV & Developer", roles: ["spv", "dev"], initials: "LG", team: "R&D" },
  { id: "zacky", name: "Zacky", role: "System Analyst & Developer", roles: ["sa", "dev"], initials: "ZK", team: "R&D" },
  { id: "mul", name: "Mul", role: "Developer", roles: ["dev"], initials: "ML", team: "R&D" },
  { id: "irfan", name: "Irfan", role: "Project Admin & Tech Writer", roles: ["admin"], initials: "IR", team: "R&D" },
  { id: "rani", name: "Rani", role: "QA Engineer", roles: ["qa"], initials: "RN", team: "QA" },
  { id: "bayu", name: "Bayu", role: "QA Engineer", roles: ["qa"], initials: "BY", team: "QA" },
];

const ROLE_FILTERS = [
  { id: "all", label: "Semua" },
  { id: "spv", label: "SPV" },
  { id: "sa", label: "SA" },
  { id: "dev", label: "Dev" },
  { id: "qa", label: "QA" },
  { id: "admin", label: "Admin" },
];

const BOARDS = [
  { id: "ibs-gen2", name: "IBS Gen 2", tag: "CBS Konvensional" },
  { id: "ibss-gen25", name: "IBSS Gen 2.5", tag: "CBS Syariah" },
  { id: "ibs-estatement", name: "IBS eStatement", tag: "Produk Pendukung" },
  { id: "ibs-branchless", name: "IBS Branchless", tag: "Produk Pendukung" },
];

const BIG_TASKS = [
  { id: "bt1", boardId: "ibs-gen2", name: "IBS Reporting", startDate: "2026-07-20", deadline: "2026-08-12", pic: "mul" },
  { id: "bt2", boardId: "ibs-gen2", name: "Penambahan Fitur CKPN", startDate: "2026-07-15", deadline: "2026-08-13", pic: "mul" },
  { id: "bt3", boardId: "ibs-gen2", name: "Revaluasi Aset Tetap", startDate: "2026-07-10", deadline: "2026-08-13", pic: "zacky" },
  { id: "bt4", boardId: "ibss-gen25", name: "Host to Host", startDate: "2026-07-18", deadline: "2026-08-13", pic: "zacky" },
  { id: "bt5", boardId: "ibs-branchless", name: "Branchless Pro", startDate: "2026-07-22", deadline: "2026-08-17", pic: "zacky" },
  { id: "bt6", boardId: "ibs-gen2", name: "Onboarding Nasabah", startDate: "2026-07-01", deadline: "2026-08-18", pic: "irfan" },
  { id: "bt7", boardId: "ibss-gen25", name: "EOM Engine", startDate: "2026-06-15", deadline: "2026-09-15", pic: "mul" },
  { id: "bt8", boardId: "ibs-estatement", name: "Modul Generate PDF", startDate: "2026-07-25", deadline: "2026-08-10", pic: "lugina" },
  { id: "bt9", boardId: "ibs-estatement", name: "Modul Notifikasi WA", startDate: "2026-08-15", deadline: "2026-09-05", pic: "irfan" },
  { id: "bt10", boardId: "ibss-gen25", name: "Migrasi Modul Zakat", startDate: "2026-06-01", deadline: "2026-08-05", pic: "zacky", onHold: true },
];

const TASK_CARDS = [
  { id: "tc1", bigTaskId: "bt1", title: "Fix query lambat rekap harian", pic: "mul", actualPct: 70, deadline: "2026-08-11", attachments: 2, reviewed: true },
  { id: "tc2", bigTaskId: "bt1", title: "Validasi format export Excel", pic: "mul", actualPct: 40, deadline: "2026-08-12", attachments: 0, reviewed: false },
  { id: "tc3", bigTaskId: "bt2", title: "Setup skema perhitungan CKPN kolektif", pic: "mul", actualPct: 10, deadline: "2026-08-09", attachments: 1, reviewed: false },
  { id: "tc4", bigTaskId: "bt3", title: "Migrasi saldo revaluasi ke OCI", pic: "zacky", actualPct: 80, deadline: "2026-08-12", attachments: 3, reviewed: true },
  { id: "tc5", bigTaskId: "bt3", title: "UAT bareng tim finance BPR", pic: "irfan", actualPct: 60, deadline: "2026-08-13", attachments: 0, reviewed: false },
  { id: "tc6", bigTaskId: "bt4", title: "Integrasi signature SNAP BI", pic: "zacky", actualPct: 80, deadline: "2026-08-11", attachments: 1, reviewed: true },
  { id: "tc7", bigTaskId: "bt5", title: "Redesign flow OTP WhatsApp", pic: "zacky", actualPct: 65, deadline: "2026-08-16", attachments: 2, reviewed: false },
  { id: "tc8", bigTaskId: "bt6", title: "Dokumentasi SOP onboarding cabang", pic: "irfan", actualPct: 68, deadline: "2026-08-17", attachments: 4, reviewed: false },
  { id: "tc9", bigTaskId: "bt7", title: "Perhitungan bunga berjalan EOM", pic: "mul", actualPct: 55, deadline: "2026-09-10", attachments: 0, reviewed: true },
  { id: "tc10", bigTaskId: "bt8", title: "Password dinamis per statement", pic: "lugina", actualPct: 100, deadline: "2026-08-08", attachments: 2, reviewed: true },
  { id: "tc11", bigTaskId: "bt8", title: "Preview PDF di portal review", pic: "lugina", actualPct: 45, deadline: "2026-08-10", attachments: 1, reviewed: false },
  { id: "tc12", bigTaskId: "bt10", title: "Mapping akun zakat ke COA", pic: "zacky", actualPct: 30, deadline: "2026-08-04", attachments: 0, reviewed: true },
];

const DAILY_TASKS_INITIAL = [
  {
    id: "dt1",
    bigTaskId: "bt1",
    title: "Fix query lambat rekap harian · Dev v1",
    pic: "mul",
    startDate: "2026-08-05",
    endDate: "2026-08-07",
    days: [
      { date: "2026-08-05", planned: "Analisis query lambat di modul rekap", done: true, blocker: "", note: "" },
      { date: "2026-08-06", planned: "Optimasi index tabel transaksi", done: true, blocker: "", note: "" },
      { date: "2026-08-07", planned: "Fix & deliver ke QA buat verifikasi", done: true, blocker: "", note: "" },
    ],
  },
  {
    id: "dt1b",
    bigTaskId: "bt1",
    title: "Fix query lambat rekap harian · Verifikasi",
    pic: "rani",
    startDate: "2026-08-08",
    endDate: "2026-08-09",
    days: [
      { date: "2026-08-08", planned: "Regression test query rekap harian", done: true, blocker: "", note: "" },
      { date: "2026-08-09", planned: "Cek edge case data volume besar", done: false, blocker: "Ada bug baru muncul di data > 50rb baris", note: "Balikin ke Mul buat fix v2" },
    ],
  },
  {
    id: "dt2",
    bigTaskId: "bt3",
    title: "Migrasi saldo revaluasi ke OCI",
    pic: "zacky",
    startDate: "2026-08-07",
    endDate: "2026-08-10",
    days: [
      { date: "2026-08-07", planned: "Setup mapping akun OCI", done: true, blocker: "", note: "" },
      { date: "2026-08-08", planned: "Migrasi data batch 1 (kejar deadline, masuk weekend)", done: true, blocker: "", note: "" },
      { date: "2026-08-09", planned: "Migrasi batch 2 + validasi", done: false, blocker: "Data batch 2 dari finance belum masuk", note: "Follow up tim finance Senin pagi" },
      { date: "2026-08-10", planned: "Rekonsiliasi akhir & sign-off", done: false, blocker: "", note: "" },
    ],
  },
  {
    id: "dt3",
    bigTaskId: "bt2",
    title: "Setup skema perhitungan CKPN kolektif",
    pic: "mul",
    startDate: "2026-08-06",
    endDate: "2026-08-08",
    days: [
      { date: "2026-08-06", planned: "Riset ketentuan SEOJK terkait CKPN kolektif", done: true, blocker: "", note: "" },
      { date: "2026-08-07", planned: "Draft skema perhitungan", done: false, blocker: "Perlu konfirmasi rumus ke tim risk", note: "" },
      { date: "2026-08-08", planned: "Review bareng SA", done: false, blocker: "", note: "" },
    ],
  },
];

const COMMENTS_INITIAL = [
  {
    id: "cm1",
    bigTaskId: "bt1",
    dailyTaskId: null,
    author: "lugina",
    text: "Ini prioritas tinggi bro, BPR NTB nanya-nanya terus soal fitur ini. Kejar ya.",
    date: "2026-08-05",
  },
  {
    id: "cm2",
    bigTaskId: "bt1",
    dailyTaskId: "dt1",
    author: "mul",
    text: "Query udah dioptimasi, index tambahan naikin performa lumayan drastis. Siap di-deliver ke QA.",
    date: "2026-08-07",
  },
  {
    id: "cm3",
    bigTaskId: "bt1",
    dailyTaskId: "dt1b",
    author: "rani",
    text: "Nemu bug pas data-nya di atas 50rb baris, query timeout. Butuh fix sebelum lanjut UAT.",
    date: "2026-08-09",
  },
];

const CHEAT_SHEET_INITIAL = [
  {
    id: "cs1",
    boardId: "ibss-gen25",
    type: "note",
    title: "Deploy Host to Host",
    value: "Worker H2H udah di-deploy di server Windows A (10.20.14.12). Silakan direview, aplikasi desktop jadi ga ada URL yang bisa diakses langsung.",
    author: "zacky",
    date: "2026-08-07",
  },
  {
    id: "cs2",
    boardId: "ibs-gen2",
    type: "url",
    title: "Repository IBS Gen 2",
    value: "https://github.com/ussi-pinbuk/ibs-gen2",
    author: "mul",
    date: "2026-08-01",
  },
  {
    id: "cs3",
    boardId: "ibs-gen2",
    type: "file",
    title: "Dokumentasi API perhitungan CKPN",
    value: "API_CKPN_v2.docx",
    author: "irfan",
    date: "2026-08-06",
  },
];


function isWeekend(dateStr) {
  const d = new Date(dateStr).getDay();
  return d === 0 || d === 6;
}

function daysBetween(a, b) {
  return Math.round((b - a) / 86400000);
}

function getWeekStart(date) {
  const d = new Date(date);
  const day = d.getDay();
  const diff = (day === 0 ? -6 : 1) - day;
  d.setDate(d.getDate() + diff);
  return d.toISOString().slice(0, 10);
}

function getWeekDates(weekStartStr) {
  const start = new Date(weekStartStr);
  const dates = [];
  for (let i = 0; i < 7; i++) {
    const d = new Date(start);
    d.setDate(d.getDate() + i);
    dates.push(d.toISOString().slice(0, 10));
  }
  return dates;
}

function shiftWeek(weekStartStr, deltaWeeks) {
  const d = new Date(weekStartStr);
  d.setDate(d.getDate() + deltaWeeks * 7);
  return d.toISOString().slice(0, 10);
}

function dailyTaskStats(dt) {
  const total = dt.days.length;
  const done = dt.days.filter((d) => d.done).length;
  const pct = total ? Math.round((done / total) * 100) : 0;
  const blockedDays = dt.days.filter((d) => !d.done && d.blocker).length;
  const daysLeft = daysBetween(TODAY, new Date(dt.endDate));
  return { total, done, pct, blockedDays, daysLeft };
}

function bigTaskStats(bt, cardsList) {
  const cards = cardsList.filter((t) => t.bigTaskId === bt.id);
  const actualPct = cards.length
    ? Math.round(cards.reduce((s, c) => s + c.actualPct, 0) / cards.length)
    : 0;
  const start = new Date(bt.startDate);
  const end = new Date(bt.deadline);
  const totalDays = Math.max(daysBetween(start, end), 1);
  const elapsed = Math.min(Math.max(daysBetween(start, TODAY), 0), totalDays);
  const expectedPct = Math.round((elapsed / totalDays) * 100);
  const daysLeft = daysBetween(TODAY, end);
  let verdict;
  if (actualPct >= 100) verdict = daysLeft >= 0 ? "win" : "lose";
  else if (daysLeft < 0) verdict = "lose";
  else verdict = actualPct >= expectedPct ? "ontrack" : "offtrack";
  const unreviewed = cards.filter((c) => !c.reviewed).length;
  let status;
  if (bt.onHold) status = "hold";
  else if (TODAY < start) status = "notstarted";
  else if (actualPct >= 100) status = "done";
  else status = "running";
  return { cards, actualPct, expectedPct, daysLeft, verdict, unreviewed, status };
}

const STATUS_LABEL = {
  notstarted: "Belum berjalan",
  running: "Sedang berjalan",
  done: "Sudah selesai",
  hold: "Di hold",
};

const VERDICT_LABEL = {
  win: "Selesai tepat waktu",
  lose: "Lewat komitmen",
  ontrack: "Sesuai jalur",
  offtrack: "Tertinggal",
  onprogress: "On progress",
};

function VerdictBadge({ verdict }) {
  return (
    <span className={`badge badge-${verdict}`}>{VERDICT_LABEL[verdict]}</span>
  );
}

function Avatar({ id, size = 26 }) {
  const person = TEAM.find((t) => t.id === id);
  if (!person) return null;
  return (
    <div
      className="avatar"
      style={{ width: size, height: size, fontSize: size * 0.4 }}
      title={person.name}
    >
      {person.initials}
    </div>
  );
}

function DualBar({ expected, actual }) {
  return (
    <div className="dualbar">
      <div className="dualbar-track">
        <div
          className={`dualbar-fill ${actual >= expected ? "fill-good" : "fill-bad"}`}
          style={{ width: `${Math.min(actual, 100)}%` }}
        />
        <div className="dualbar-tick" style={{ left: `${Math.min(expected, 100)}%` }} />
      </div>
      <div className="dualbar-legend">
        <span className="mono">{actual}%</span>
        <span className="dualbar-legend-muted">target {expected}%</span>
      </div>
    </div>
  );
}

function StatCard({ label, value, tone }) {
  return (
    <div className={`stat-card ${tone ? `tone-${tone}` : ""}`}>
      <div className="stat-label">{label}</div>
      <div className="stat-value mono">{value}</div>
    </div>
  );
}

function BreakdownBar({ label, value, max, colorClass }) {
  return (
    <div className="breakdown-row">
      <span className="breakdown-label">{label}</span>
      <div className="breakdown-track">
        <div className={`breakdown-fill ${colorClass}`} style={{ width: `${max ? (value / max) * 100 : 0}%` }} />
      </div>
      <span className="mono breakdown-value">{value}</span>
    </div>
  );
}

function Dashboard({ cards, onOpenTask }) {
  const allStats = BIG_TASKS.map((bt) => ({ bt, ...bigTaskStats(bt, cards) }));
  const total = allStats.length;
  const notStarted = allStats.filter((s) => s.status === "notstarted").length;
  const running = allStats.filter((s) => s.status === "running").length;
  const done = allStats.filter((s) => s.status === "done").length;
  const hold = allStats.filter((s) => s.status === "hold").length;
  const won = allStats.filter((s) => s.verdict === "win").length;
  const lose = allStats.filter((s) => s.verdict === "lose").length;
  const completionRate = total ? Math.round((done / total) * 100) : 0;
  const totalUnreviewed = cards.filter((c) => !c.reviewed).length;

  const runningItems = allStats.filter((s) => s.status === "running");
  const loseItems = allStats.filter((s) => s.verdict === "lose");
  const nearestDeadline = [...runningItems].sort((a, b) => a.daysLeft - b.daysLeft).slice(0, 5);

  const statusChartData = [
    { name: "Sedang berjalan", value: running, color: "#3E7D6E" },
    { name: "Belum berjalan", value: notStarted, color: "#5F6B72" },
    { name: "Sudah selesai", value: done, color: "#34506E" },
    { name: "Di hold", value: hold, color: "#9E8A5C" },
  ];
  const hasilChartData = [
    { name: "Won", value: won, color: "#3E7D6E" },
    { name: "Lose", value: lose, color: "#B85C52" },
  ];
  const progressChartData = runningItems.map(({ bt, actualPct, expectedPct }) => ({
    name: bt.name.length > 12 ? bt.name.slice(0, 11) + "…" : bt.name,
    actual: actualPct,
    expected: expectedPct,
  }));

  return (
    <div>
      <div className="dash-heading">
        <span className="dash-heading-title">Project tracking dashboard — R&amp;D</span>
        <span className="muted small">Live dari semua board, big task, dan task card</span>
      </div>

      <div className="stat-row stat-row-8">
        <StatCard label="Total big task" value={total} />
        <StatCard label="Belum berjalan" value={notStarted} />
        <StatCard label="Sedang berjalan" value={running} />
        <StatCard label="Sudah selesai" value={done} />
        <StatCard label="Di hold" value={hold} />
        <StatCard label="Won" value={won} tone="good" />
        <StatCard label="Lose" value={lose} tone="warn" />
        <StatCard label="Completion rate" value={`${completionRate}%`} tone="accent" />
      </div>

      <div className="two-col">
        <div className="section">
          <div className="section-title">Status project</div>
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <div style={{ width: 130, height: 130, flexShrink: 0 }}>
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={statusChartData}
                    dataKey="value"
                    nameKey="name"
                    cx="50%"
                    cy="50%"
                    innerRadius={28}
                    outerRadius={58}
                    stroke="#FFFFFF"
                    strokeWidth={1}
                  >
                    {statusChartData.map((d, i) => (
                      <Cell key={i} fill={d.color} />
                    ))}
                  </Pie>
                  <Tooltip contentStyle={{ fontFamily: "Tahoma, sans-serif", fontSize: 11 }} />
                </PieChart>
              </ResponsiveContainer>
            </div>
            <div className="chart-legend">
              {statusChartData.map((d) => (
                <div className="chart-legend-row" key={d.name}>
                  <span className="chart-legend-swatch" style={{ background: d.color }} />
                  <span className="small">{d.name}</span>
                  <span className="mono small" style={{ marginLeft: "auto" }}>{d.value}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
        <div className="section">
          <div className="section-title">Hasil project</div>
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <div style={{ width: 130, height: 130, flexShrink: 0 }}>
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={hasilChartData}
                    dataKey="value"
                    nameKey="name"
                    cx="50%"
                    cy="50%"
                    innerRadius={28}
                    outerRadius={58}
                    stroke="#FFFFFF"
                    strokeWidth={1}
                  >
                    {hasilChartData.map((d, i) => (
                      <Cell key={i} fill={d.color} />
                    ))}
                  </Pie>
                  <Tooltip contentStyle={{ fontFamily: "Tahoma, sans-serif", fontSize: 11 }} />
                </PieChart>
              </ResponsiveContainer>
            </div>
            <div className="chart-legend">
              {hasilChartData.map((d) => (
                <div className="chart-legend-row" key={d.name}>
                  <span className="chart-legend-swatch" style={{ background: d.color }} />
                  <span className="small">{d.name}</span>
                  <span className="mono small" style={{ marginLeft: "auto" }}>{d.value}</span>
                </div>
              ))}
              <div className="empty-note" style={{ marginTop: 6 }}>
                {totalUnreviewed} task card belum lo review.
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="section">
        <div className="section-title">Progress: actual vs expected (%)</div>
        <div style={{ width: "100%", height: 220 }}>
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={progressChartData} margin={{ top: 8, right: 8, left: -18, bottom: 8 }}>
              <CartesianGrid stroke="#D6DADD" vertical={false} />
              <XAxis
                dataKey="name"
                tick={{ fontFamily: "Tahoma, sans-serif", fontSize: 10, fill: "#000000" }}
                axisLine={{ stroke: "#808080" }}
                tickLine={false}
              />
              <YAxis
                domain={[0, 100]}
                tick={{ fontFamily: "Tahoma, sans-serif", fontSize: 10, fill: "#000000" }}
                axisLine={{ stroke: "#808080" }}
                tickLine={false}
              />
              <Tooltip contentStyle={{ fontFamily: "Tahoma, sans-serif", fontSize: 11 }} />
              <Legend wrapperStyle={{ fontFamily: "Tahoma, sans-serif", fontSize: 11 }} />
              <Bar dataKey="expected" name="Expected %" fill="#AEB6BC" />
              <Bar dataKey="actual" name="Actual %" fill="#34506E" />
            </BarChart>
          </ResponsiveContainer>
        </div>
        <div className="attention-list" style={{ marginTop: 10 }}>
          {runningItems.map(({ bt, verdict, expectedPct, actualPct, daysLeft }) => (
            <div className="attention-row" key={bt.id} onClick={() => {}}>
              <div className="attention-main">
                <span className="attention-name">{bt.name}</span>
                <span className="muted small">{BOARDS.find((b) => b.id === bt.boardId)?.name}</span>
              </div>
              <div className="attention-bar">
                <DualBar expected={expectedPct} actual={actualPct} />
              </div>
              <VerdictBadge verdict={verdict} />
              <span className={`mono small ${daysLeft < 0 ? "days-late" : "muted"}`}>
                {daysLeft < 0 ? `telat ${Math.abs(daysLeft)}h` : `${daysLeft}h lagi`}
              </span>
            </div>
          ))}
          {runningItems.length === 0 && <div className="empty-note">Tidak ada project yang sedang berjalan.</div>}
        </div>
      </div>

      <div className="two-col">
        <div className="section">
          <div className="section-title">Deadline terdekat</div>
          <table className="sheet-table">
            <thead>
              <tr><th>Nama project</th><th>PIC</th><th>Sisa hari</th><th>Actual</th></tr>
            </thead>
            <tbody>
              {nearestDeadline.map(({ bt, daysLeft, actualPct }) => (
                <tr key={bt.id}>
                  <td>{bt.name}</td>
                  <td><Avatar id={bt.pic} size={18} /></td>
                  <td className={`mono ${daysLeft < 0 ? "days-late" : ""}`}>{daysLeft}</td>
                  <td className="mono">{actualPct}%</td>
                </tr>
              ))}
              {nearestDeadline.length === 0 && (
                <tr><td colSpan="4" className="empty-note">Tidak ada deadline mendatang.</td></tr>
              )}
            </tbody>
          </table>
        </div>
        <div className="section">
          <div className="section-title">Berstatus lose — perlu perhatian</div>
          <table className="sheet-table">
            <thead>
              <tr><th>Nama project</th><th>Actual</th><th>Expected</th><th>Sisa hari</th></tr>
            </thead>
            <tbody>
              {loseItems.map(({ bt, actualPct, expectedPct, daysLeft }) => (
                <tr key={bt.id}>
                  <td>{bt.name}</td>
                  <td className="mono">{actualPct}%</td>
                  <td className="mono">{expectedPct}%</td>
                  <td className={`mono ${daysLeft < 0 ? "days-late" : ""}`}>{daysLeft}</td>
                </tr>
              ))}
              {loseItems.length === 0 && (
                <tr><td colSpan="4" className="empty-note">Tidak ada project berstatus lose saat ini.</td></tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      <div className="section">
        <div className="section-title">Ringkasan tim</div>
        <div className="team-grid">
          {TEAM.map((person) => {
            const personCards = cards.filter((c) => c.pic === person.id);
            const active = personCards.filter((c) => c.actualPct < 100).length;
            const unreviewed = personCards.filter((c) => !c.reviewed).length;
            return (
              <div className="team-card" key={person.id}>
                <div className="team-card-top">
                  <Avatar id={person.id} size={32} />
                  <div>
                    <div className="team-name">{person.name}</div>
                    <div className="muted small">{person.role}</div>
                  </div>
                </div>
                <div className="team-card-stats">
                  <span><span className="mono">{active}</span> task jalan</span>
                  <span className={unreviewed ? "accent-text" : "muted"}>
                    <span className="mono">{unreviewed}</span> belum direview
                  </span>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

function AddBigTaskForm({ onAdd, onCancel }) {
  const [name, setName] = useState("");
  return (
    <div className="inline-form">
      <input
        className="inline-input"
        placeholder="Nama big task, misal: Tahap Analisis"
        value={name}
        onChange={(e) => setName(e.target.value)}
        autoFocus
      />
      <button
        className="quick-btn quick-btn-done"
        onClick={() => {
          if (name.trim()) onAdd(name.trim());
        }}
      >
        <Check size={11} aria-hidden="true" /> Simpan
      </button>
      <button className="quick-btn" onClick={onCancel}>
        <X size={11} aria-hidden="true" />
      </button>
    </div>
  );
}

function AddDailyTaskForm({ onAdd, onCancel, defaultPic, defaultTitle }) {
  const [title, setTitle] = useState(defaultTitle || "");
  const [start, setStart] = useState("2026-08-08");
  const [end, setEnd] = useState("2026-08-08");
  const [roleFilter, setRoleFilter] = useState("all");
  const filteredTeam = TEAM.filter((t) => roleFilter === "all" || t.roles.includes(roleFilter));
  const [pic, setPic] = useState(defaultPic || TEAM[0].id);

  return (
    <div className="inline-form inline-form-daily">
      <input
        className="inline-input"
        placeholder="Nama daily task, misal: Task A · Verifikasi"
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        autoFocus
      />
      <div className="inline-form-row">
        <label className="small muted">PIC</label>
        <div className="role-filter-pills">
          {ROLE_FILTERS.map((r) => (
            <button
              key={r.id}
              className={`role-filter-pill ${roleFilter === r.id ? "role-filter-pill-active" : ""}`}
              onClick={() => setRoleFilter(r.id)}
            >
              {r.label}
            </button>
          ))}
        </div>
      </div>
      <select className="inline-input" value={pic} onChange={(e) => setPic(e.target.value)}>
        {filteredTeam.map((t) => (
          <option key={t.id} value={t.id}>
            {t.name} — {t.role} {t.team === "QA" ? "(tim QA)" : ""}
          </option>
        ))}
      </select>
      <div className="inline-form-dates">
        <label className="small muted">Mulai</label>
        <input type="date" className="inline-input" value={start} onChange={(e) => setStart(e.target.value)} />
        <label className="small muted">Selesai</label>
        <input type="date" className="inline-input" value={end} onChange={(e) => setEnd(e.target.value)} />
      </div>
      <div className="inline-form-actions">
        <button
          className="quick-btn quick-btn-done"
          onClick={() => {
            if (title.trim() && start && end && start <= end) onAdd(title.trim(), start, end, pic);
          }}
        >
          <Check size={11} aria-hidden="true" /> Buat {daysBetween(new Date(start), new Date(end)) + 1} baris harian
        </button>
        <button className="quick-btn" onClick={onCancel}>
          <X size={11} aria-hidden="true" />
        </button>
      </div>
    </div>
  );
}

function DailyTaskCard({ dt, onUpdateDay, onToggleDone, onCommentClick, onReviewClick }) {
  const stats = dailyTaskStats(dt);
  const person = TEAM.find((t) => t.id === dt.pic);
  return (
    <div className="daily-task-card">
      <div className="daily-task-head">
        <div>
          <span className="daily-task-title">{dt.title}</span>
          <span className="muted small" style={{ marginLeft: 8 }}>
            {dt.startDate} → {dt.endDate}
          </span>
        </div>
        <div className="daily-task-head-right">
          {person?.team === "QA" && <span className="team-tag">Tim QA</span>}
          <Avatar id={dt.pic} size={20} />
          <span className="small">{person?.name}</span>
          <span className="mono small">{stats.done}/{stats.total} hari</span>
          <span className="mono small accent-text">{stats.pct}%</span>
          <button className="comment-jump-btn" onClick={() => onCommentClick(dt.id)}>
            <MessageSquare size={12} aria-hidden="true" /> Komentar
          </button>
          <div className="review-clone-group">
            <span className="muted small">Review:</span>
            <button className="review-clone-btn" onClick={() => onReviewClick(dt, "SPV")}>SPV</button>
            <button className="review-clone-btn" onClick={() => onReviewClick(dt, "QA")}>QA</button>
          </div>
        </div>
      </div>
      <table className="sheet-table daily-day-table">
        <thead>
          <tr>
            <th>Tanggal</th>
            <th>Rencana</th>
            <th>Status</th>
            <th>Blocker / catatan lanjutan</th>
          </tr>
        </thead>
        <tbody>
          {dt.days.map((d) => {
            const weekend = isWeekend(d.date);
            return (
              <tr key={d.date} className={weekend ? "row-weekend" : ""}>
                <td className="mono small">
                  {d.date}
                  {weekend && <span className="lembur-badge">lembur</span>}
                </td>
                <td>
                  <input
                    className="inline-input inline-input-cell"
                    value={d.planned}
                    placeholder="(belum diisi)"
                    onChange={(e) => onUpdateDay(dt.id, d.date, "planned", e.target.value)}
                  />
                </td>
                <td>
                  <button
                    className={`day-status-btn ${d.done ? "day-status-done" : "day-status-open"}`}
                    onClick={() => onToggleDone(dt.id, d.date)}
                  >
                    {d.done ? "Selesai" : "Belum"}
                  </button>
                </td>
                <td>
                  {!d.done ? (
                    <input
                      className="inline-input inline-input-cell"
                      value={d.blocker}
                      placeholder="Blocker / rencana lanjut..."
                      onChange={(e) => onUpdateDay(dt.id, d.date, "blocker", e.target.value)}
                    />
                  ) : (
                    <span className="muted small">—</span>
                  )}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

const CURRENT_USER = "lugina";

function renderCommentText(text) {
  const mentionSet = new Set(TEAM.map((t) => "@" + t.name));
  const names = TEAM.map((t) => t.name).join("|");
  const regex = new RegExp(`(@(?:${names}))`, "g");
  const parts = text.split(regex);
  return parts.map((part, i) =>
    mentionSet.has(part) ? (
      <span className="mention-tag" key={i}>{part}</span>
    ) : (
      part
    )
  );
}

function CheatSheetSection({ boardId, items, onAdd }) {
  const [showAdd, setShowAdd] = useState(false);
  const [type, setType] = useState("note");
  const [title, setTitle] = useState("");
  const [value, setValue] = useState("");

  const typeIcon = (t) => {
    if (t === "file") return <File size={13} aria-hidden="true" />;
    if (t === "url") return <Link2 size={13} aria-hidden="true" />;
    return <StickyNote size={13} aria-hidden="true" />;
  };

  const reset = () => {
    setTitle("");
    setValue("");
    setType("note");
    setShowAdd(false);
  };

  return (
    <div className="cheatsheet-section">
      <div className="section-title">Cheat sheet / referensi board</div>
      <div className="cheatsheet-list">
        {items.map((it) => {
          const person = TEAM.find((t) => t.id === it.author);
          return (
            <div className="cheatsheet-row" key={it.id}>
              <div className="cheatsheet-icon">{typeIcon(it.type)}</div>
              <div className="cheatsheet-main">
                <div className="cheatsheet-title">{it.title}</div>
                {it.type === "url" ? (
                  <a className="cheatsheet-link" href={it.value} target="_blank" rel="noreferrer">
                    {it.value}
                  </a>
                ) : it.type === "file" ? (
                  <span className="muted small">{it.value}</span>
                ) : (
                  <span className="cheatsheet-note">{it.value}</span>
                )}
              </div>
              <div className="cheatsheet-meta">
                <Avatar id={it.author} size={18} />
                <span className="small">{person?.name}</span>
                <span className="mono small muted">{it.date}</span>
              </div>
            </div>
          );
        })}
        {items.length === 0 && <div className="empty-note">Belum ada referensi buat board ini.</div>}
      </div>

      {showAdd ? (
        <div className="inline-form inline-form-daily">
          <div className="role-filter-pills">
            <button className={`role-filter-pill ${type === "note" ? "role-filter-pill-active" : ""}`} onClick={() => setType("note")}>
              <StickyNote size={11} aria-hidden="true" /> Catatan
            </button>
            <button className={`role-filter-pill ${type === "url" ? "role-filter-pill-active" : ""}`} onClick={() => setType("url")}>
              <Link2 size={11} aria-hidden="true" /> URL
            </button>
            <button className={`role-filter-pill ${type === "file" ? "role-filter-pill-active" : ""}`} onClick={() => setType("file")}>
              <File size={11} aria-hidden="true" /> File
            </button>
          </div>
          <input
            className="inline-input"
            placeholder="Judul, misal: Deploy Host to Host"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
          />
          {type === "note" && (
            <textarea
              className="inline-input inline-textarea"
              placeholder="Tulis keterangan lengkapnya..."
              value={value}
              onChange={(e) => setValue(e.target.value)}
            />
          )}
          {type === "url" && (
            <input
              className="inline-input"
              placeholder="https://..."
              value={value}
              onChange={(e) => setValue(e.target.value)}
            />
          )}
          {type === "file" && (
            <label className="file-upload-btn">
              <Upload size={12} aria-hidden="true" />
              {value || "Pilih file..."}
              <input
                type="file"
                style={{ display: "none" }}
                onChange={(e) => setValue(e.target.files?.[0]?.name || "")}
              />
            </label>
          )}
          <div className="inline-form-actions">
            <button
              className="quick-btn quick-btn-done"
              onClick={() => {
                if (!title.trim() || !value.trim()) return;
                onAdd(boardId, type, title.trim(), value.trim());
                reset();
              }}
            >
              <Check size={11} aria-hidden="true" /> Simpan
            </button>
            <button className="quick-btn" onClick={reset}>
              <X size={11} aria-hidden="true" />
            </button>
          </div>
        </div>
      ) : (
        <button className="add-card-ghost" onClick={() => setShowAdd(true)}>
          <Plus size={13} aria-hidden="true" /> Tambah referensi
        </button>
      )}
    </div>
  );
}

function CommentSection({ bigTaskId, dailyTasksForBigTask, comments, onAddComment, jumpScope, jumpToken }) {
  const [filterScope, setFilterScope] = useState("all");
  const [composeScope, setComposeScope] = useState("general");
  const [text, setText] = useState("");
  const [showMentions, setShowMentions] = useState(false);
  const sectionRef = useRef(null);
  const inputRef = useRef(null);

  useEffect(() => {
    if (jumpScope) {
      setFilterScope(jumpScope);
      setComposeScope(jumpScope);
      sectionRef.current?.scrollIntoView({ behavior: "smooth", block: "nearest" });
      inputRef.current?.focus();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [jumpToken]);

  const mentionMatch = text.match(/@(\w*)$/);
  const mentionQuery = mentionMatch ? mentionMatch[1].toLowerCase() : null;
  const mentionSuggestions =
    mentionQuery !== null ? TEAM.filter((t) => t.name.toLowerCase().startsWith(mentionQuery)) : [];

  const pickMention = (name) => {
    setText((t) => t.replace(/@(\w*)$/, `@${name} `));
    inputRef.current?.focus();
  };

  const scopedComments = comments.filter((c) => {
    if (filterScope === "all") return true;
    if (filterScope === "general") return !c.dailyTaskId;
    return c.dailyTaskId === filterScope;
  });
  const sorted = [...scopedComments].sort((a, b) => (a.date < b.date ? 1 : -1));

  const scopeLabel = (dailyTaskId) => {
    if (!dailyTaskId) return "Umum";
    const dt = dailyTasksForBigTask.find((d) => d.id === dailyTaskId);
    return dt ? dt.title : "Umum";
  };

  return (
    <div className="comment-section" ref={sectionRef}>
      <div className="section-title">Komentar</div>
      <div className="comment-filter-row">
        <button
          className={`role-filter-pill ${filterScope === "all" ? "role-filter-pill-active" : ""}`}
          onClick={() => setFilterScope("all")}
        >
          Semua
        </button>
        <button
          className={`role-filter-pill ${filterScope === "general" ? "role-filter-pill-active" : ""}`}
          onClick={() => setFilterScope("general")}
        >
          Umum
        </button>
        {dailyTasksForBigTask.map((dt) => (
          <button
            key={dt.id}
            className={`role-filter-pill ${filterScope === dt.id ? "role-filter-pill-active" : ""}`}
            onClick={() => setFilterScope(dt.id)}
          >
            {dt.title}
          </button>
        ))}
      </div>

      <div className="comment-list">
        {sorted.map((c) => {
          const person = TEAM.find((t) => t.id === c.author);
          return (
            <div className="comment-row" key={c.id}>
              <Avatar id={c.author} size={22} />
              <div className="comment-body">
                <div className="comment-meta">
                  <span className="comment-author">{person?.name}</span>
                  <span className="comment-scope-tag">{scopeLabel(c.dailyTaskId)}</span>
                  <span className="mono small muted">{c.date}</span>
                </div>
                <div className="comment-text">{renderCommentText(c.text)}</div>
              </div>
            </div>
          );
        })}
        {sorted.length === 0 && <div className="empty-note">Belum ada komentar di scope ini.</div>}
      </div>

      <div className="comment-composer">
        <div className="comment-composer-row">
          <select className="inline-input" value={composeScope} onChange={(e) => setComposeScope(e.target.value)}>
            <option value="general">Umum (level big task)</option>
            {dailyTasksForBigTask.map((dt) => (
              <option key={dt.id} value={dt.id}>{dt.title}</option>
            ))}
          </select>
        </div>
        <div className="comment-composer-row comment-input-wrap">
          <div className="comment-input-relative">
            <input
              ref={inputRef}
              className="inline-input"
              placeholder="Tulis komentar... ketik @ buat mention"
              value={text}
              onChange={(e) => setText(e.target.value)}
              onFocus={() => setShowMentions(true)}
              onBlur={() => setTimeout(() => setShowMentions(false), 150)}
            />
            {showMentions && mentionQuery !== null && mentionSuggestions.length > 0 && (
              <div className="mention-popup">
                {mentionSuggestions.map((t) => (
                  <button
                    key={t.id}
                    className="mention-option"
                    onMouseDown={(e) => e.preventDefault()}
                    onClick={() => pickMention(t.name)}
                  >
                    <Avatar id={t.id} size={18} />
                    <span>{t.name}</span>
                    <span className="muted small">{t.role}</span>
                  </button>
                ))}
              </div>
            )}
          </div>
          <button
            className="quick-btn quick-btn-done"
            onClick={() => {
              if (!text.trim()) return;
              onAddComment(bigTaskId, composeScope === "general" ? null : composeScope, CURRENT_USER, text.trim());
              setText("");
            }}
          >
            <Check size={11} aria-hidden="true" /> Kirim
          </button>
        </div>
      </div>
    </div>
  );
}

function ProjectView({
  board,
  boardBigTasks,
  dailyTasks,
  onAddDailyTask,
  onUpdateDay,
  onToggleDone,
  onAddBigTask,
  comments,
  onAddComment,
  cheatSheet,
  onAddCheatSheet,
  signedBigTasks,
  onSignBigTask,
  onUnsignBigTask,
}) {
  const [selectedBt, setSelectedBt] = useState(boardBigTasks[0]?.id || null);
  const [showAddBigTask, setShowAddBigTask] = useState(false);
  const [showAddDaily, setShowAddDaily] = useState(false);
  const [dailyFormDefaults, setDailyFormDefaults] = useState({ title: "", pic: null });
  const [jumpScope, setJumpScope] = useState(null);
  const [jumpToken, setJumpToken] = useState(0);

  const handleCommentClick = (dailyTaskId) => {
    setJumpScope(dailyTaskId);
    setJumpToken((t) => t + 1);
  };

  const handleReviewClick = (dt, roleTag) => {
    const tag = roleTag === "SPV" ? "[Review SPV]" : "[Review QA]";
    const defaultPic = roleTag === "SPV" ? "lugina" : "rani";
    setDailyFormDefaults({ title: `${tag} ${dt.title}`, pic: defaultPic });
    setShowAddDaily(true);
  };

  const openAddDaily = () => {
    const bt = boardBigTasks.find((b) => b.id === selectedBt);
    setDailyFormDefaults({ title: "", pic: bt?.pic || null });
    setShowAddDaily(true);
  };

  const rollups = boardBigTasks.map((bt) => {
    const dts = dailyTasks.filter((d) => d.bigTaskId === bt.id);
    const pct = dts.length
      ? Math.round(dts.reduce((s, d) => s + dailyTaskStats(d).pct, 0) / dts.length)
      : 0;
    const start = new Date(bt.startDate);
    const end = new Date(bt.deadline);
    const totalDays = Math.max(daysBetween(start, end), 1);
    const elapsed = Math.min(Math.max(daysBetween(start, TODAY), 0), totalDays);
    const expectedPct = Math.round((elapsed / totalDays) * 100);
    const daysLeft = daysBetween(TODAY, end);
    const signed = !!signedBigTasks[bt.id];
    let verdict;
    if (signed) verdict = daysLeft >= 0 ? "win" : "lose";
    else if (daysLeft < 0) verdict = "lose";
    else verdict = "onprogress";
    return { bt, pct, expectedPct, daysLeft, verdict, count: dts.length, signed };
  });
  const totalBt = rollups.length;
  const doneBt = rollups.filter((r) => r.signed).length;
  const boardCompletion = totalBt ? Math.round(rollups.reduce((s, r) => s + r.pct, 0) / totalBt) : 0;
  const needsAttention = rollups.filter((r) => r.count === 0).length;
  const wonBt = rollups.filter((r) => r.verdict === "win").length;
  const loseBt = rollups.filter((r) => r.verdict === "lose").length;
  const projectDone = totalBt > 0 && doneBt === totalBt;

  const activeBt = boardBigTasks.find((bt) => bt.id === selectedBt);
  const activeDts = dailyTasks.filter((d) => d.bigTaskId === selectedBt);
  const activeRollup = rollups.find((r) => r.bt.id === selectedBt);

  return (
    <div>
      <div className="project-status-banner">
        <span className={`project-status-dot ${projectDone ? "status-done" : "status-progress"}`} />
        <span className="small">
          Status project: <strong>{projectDone ? "Selesai (semua big task sign-off)" : "Berjalan"}</strong>
        </span>
        <span className="muted small">{doneBt}/{totalBt} big task sign-off</span>
      </div>
      <div className="stat-row" style={{ gridTemplateColumns: "repeat(6, 1fr)" }}>
        <StatCard label="Total big task" value={totalBt} />
        <StatCard label="Big task selesai" value={doneBt} tone="good" />
        <StatCard label="Won" value={wonBt} tone="good" />
        <StatCard label="Lose" value={loseBt} tone="warn" />
        <StatCard label="Completion rate" value={`${boardCompletion}%`} tone="accent" />
        <StatCard label="Belum ada daily task" value={needsAttention} tone={needsAttention ? "warn" : undefined} />
      </div>

      <CheatSheetSection
        boardId={board.id}
        items={cheatSheet.filter((c) => c.boardId === board.id)}
        onAdd={onAddCheatSheet}
      />

      <div className="project-layout">
        <div className="bigtask-list">
          <div className="section-title" style={{ marginBottom: 8 }}>Big task</div>
          {rollups.map(({ bt, pct, verdict, daysLeft }) => (
            <button
              key={bt.id}
              className={`bigtask-list-item ${selectedBt === bt.id ? "bigtask-list-item-active" : ""}`}
              onClick={() => setSelectedBt(bt.id)}
            >
              <div className="bigtask-list-top">
                <span className="bigtask-list-name">{bt.name}</span>
                <VerdictBadge verdict={verdict} />
              </div>
              <div className="dualbar-track" style={{ marginTop: 4 }}>
                <div className="dualbar-fill fill-good" style={{ width: `${pct}%` }} />
              </div>
              <div className="bigtask-list-bottom">
                <span className="mono small muted">{pct}%</span>
                <span className={`mono small ${daysLeft < 0 ? "days-late" : "muted"}`}>
                  {daysLeft < 0 ? `telat ${Math.abs(daysLeft)}h` : `${daysLeft}h lagi`}
                </span>
              </div>
            </button>
          ))}
          {showAddBigTask ? (
            <AddBigTaskForm
              onAdd={(name) => {
                onAddBigTask(board.id, name);
                setShowAddBigTask(false);
              }}
              onCancel={() => setShowAddBigTask(false)}
            />
          ) : (
            <button className="add-card-ghost" onClick={() => setShowAddBigTask(true)}>
              <Plus size={13} aria-hidden="true" /> Tambah big task
            </button>
          )}
        </div>

        <div className="bigtask-detail">
          {activeBt ? (
            <>
              <div className="section-title bigtask-detail-title">
                <span>{activeBt.name} — daily task</span>
                <div className="bigtask-detail-title-right">
                  {activeRollup && <VerdictBadge verdict={activeRollup.verdict} />}
                  {activeRollup?.signed ? (
                    <button className="sign-btn sign-btn-active" onClick={() => onUnsignBigTask(activeBt.id)}>
                      <Check size={11} aria-hidden="true" /> Signed done
                    </button>
                  ) : (
                    <button
                      className="sign-btn"
                      disabled={activeRollup && activeRollup.pct < 100}
                      title={activeRollup && activeRollup.pct < 100 ? "Progress belum 100%" : ""}
                      onClick={() => onSignBigTask(activeBt.id)}
                    >
                      Tandai selesai (SPV)
                    </button>
                  )}
                </div>
              </div>
              {activeDts.map((dt) => (
                <DailyTaskCard
                  key={dt.id}
                  dt={dt}
                  onUpdateDay={onUpdateDay}
                  onToggleDone={onToggleDone}
                  onCommentClick={handleCommentClick}
                  onReviewClick={handleReviewClick}
                />
              ))}
              {activeDts.length === 0 && (
                <div className="empty-note">Belum ada daily task buat big task ini.</div>
              )}
              {showAddDaily ? (
                <AddDailyTaskForm
                  defaultPic={dailyFormDefaults.pic || activeBt.pic}
                  defaultTitle={dailyFormDefaults.title}
                  onAdd={(title, start, end, pic) => {
                    onAddDailyTask(activeBt.id, title, start, end, pic);
                    setShowAddDaily(false);
                  }}
                  onCancel={() => setShowAddDaily(false)}
                />
              ) : (
                <button className="add-card-ghost" onClick={openAddDaily}>
                  <Plus size={13} aria-hidden="true" /> Tambah daily task
                </button>
              )}
              <CommentSection
                bigTaskId={activeBt.id}
                dailyTasksForBigTask={activeDts}
                comments={comments.filter((c) => c.bigTaskId === activeBt.id)}
                onAddComment={onAddComment}
                jumpScope={jumpScope}
                jumpToken={jumpToken}
              />
            </>
          ) : (
            <div className="empty-note">Pilih atau tambahkan big task dulu.</div>
          )}
        </div>
      </div>
    </div>
  );
}

function BoardsView({
  activeBoard,
  setActiveBoard,
  dailyTasks,
  onAddDailyTask,
  onUpdateDay,
  onToggleDone,
  onAddBigTask,
  extraBigTasks,
  comments,
  onAddComment,
  cheatSheet,
  onAddCheatSheet,
  signedBigTasks,
  onSignBigTask,
  onUnsignBigTask,
}) {
  const boardBigTasks = BIG_TASKS.filter((bt) => bt.boardId === activeBoard).concat(
    extraBigTasks.filter((bt) => bt.boardId === activeBoard)
  );
  const board = BOARDS.find((b) => b.id === activeBoard);
  return (
    <div>
      <div className="board-pills-row">
        <div className="board-pills">
          {BOARDS.map((b) => (
            <button
              key={b.id}
              className={`board-pill ${activeBoard === b.id ? "board-pill-active" : ""}`}
              onClick={() => setActiveBoard(b.id)}
            >
              {b.name}
            </button>
          ))}
          <button className="board-pill board-pill-ghost">
            <Plus size={13} aria-hidden="true" /> Board baru
          </button>
        </div>
      </div>

      <ProjectView
        board={board}
        boardBigTasks={boardBigTasks}
        dailyTasks={dailyTasks}
        onAddDailyTask={onAddDailyTask}
        onUpdateDay={onUpdateDay}
        onToggleDone={onToggleDone}
        onAddBigTask={onAddBigTask}
        comments={comments}
        onAddComment={onAddComment}
        cheatSheet={cheatSheet}
        onAddCheatSheet={onAddCheatSheet}
        signedBigTasks={signedBigTasks}
        onSignBigTask={onSignBigTask}
        onUnsignBigTask={onUnsignBigTask}
      />
    </div>
  );
}

function WeeklyPlanView({ allBigTasks, dailyTasks, weekStart, onPrevWeek, onNextWeek, weeklyPushState, onPushWeekly }) {
  const weekDates = getWeekDates(weekStart);
  const weekEnd = weekDates[6];
  const todayStr = TODAY.toISOString().slice(0, 10);

  const rows = allBigTasks
    .map((bt) => {
      const dts = dailyTasks.filter((d) => d.bigTaskId === bt.id);
      let totalWeekDays = 0;
      let doneWeekDays = 0;
      let elapsedWeekDays = 0;
      dts.forEach((dt) => {
        dt.days.forEach((d) => {
          if (weekDates.includes(d.date)) {
            totalWeekDays += 1;
            if (d.done) doneWeekDays += 1;
            if (d.date <= todayStr) elapsedWeekDays += 1;
          }
        });
      });
      const board = BOARDS.find((b) => b.id === bt.boardId);
      const actualPct = totalWeekDays ? Math.round((doneWeekDays / totalWeekDays) * 100) : 0;
      const expectedPct = totalWeekDays ? Math.round((elapsedWeekDays / totalWeekDays) * 100) : 0;
      return { bt, board, totalWeekDays, doneWeekDays, actualPct, expectedPct };
    })
    .filter((r) => r.totalWeekDays > 0);

  return (
    <div>
      <div className="weekplan-header">
        <button className="quick-btn" onClick={onPrevWeek}>&larr; Minggu lalu</button>
        <span className="weekplan-range">{weekStart} → {weekEnd}</span>
        <button className="quick-btn" onClick={onNextWeek}>Minggu depan &rarr;</button>
      </div>
      <div className="empty-note" style={{ marginBottom: 10 }}>
        Rangkuman ini dihitung dari baris harian di Daily Task yang jatuh di rentang minggu ini. Push ke HR bisa
        diulang tiap hari (upsert) — callback ID tetap sama, cuma timestamp yang update.
      </div>
      <table className="sheet-table weekplan-table">
        <thead>
          <tr>
            <th>Board</th>
            <th>Big task (topic)</th>
            <th>Actual</th>
            <th>Expected</th>
            <th>Push to HR</th>
            <th>Last pushed</th>
            <th>Callback ID</th>
          </tr>
        </thead>
        <tbody>
          {rows.map(({ bt, board, actualPct, expectedPct }) => {
            const key = `${bt.id}_${weekStart}`;
            const pushInfo = weeklyPushState[key];
            return (
              <tr key={bt.id}>
                <td className="muted small">{board?.name}</td>
                <td>{bt.name}</td>
                <td className="mono">{actualPct}%</td>
                <td className="mono">{expectedPct}%</td>
                <td>
                  <button className="quick-btn quick-btn-done" onClick={() => onPushWeekly(bt.id, weekStart)}>
                    <Check size={11} aria-hidden="true" /> {pushInfo ? "Push ulang" : "Push"}
                  </button>
                </td>
                <td className="mono small">{pushInfo ? pushInfo.pushedAt : "—"}</td>
                <td className="mono small muted">{pushInfo ? pushInfo.callbackId : "—"}</td>
              </tr>
            );
          })}
          {rows.length === 0 && (
            <tr><td colSpan="7" className="empty-note">Tidak ada daily task di minggu ini.</td></tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

function ReviewQueue({ cards, reviewedState, setReviewedState, onOpenTask }) {
  const unreviewed = cards.filter((c) => !reviewedState[c.id]);
  return (
    <div className="section">
      <div className="section-title">
        Antrean review <span className="muted small">— {unreviewed.length} task menunggu atensi lo</span>
      </div>
      <div className="queue-list">
        {unreviewed.map((c) => {
          const bt = BIG_TASKS.find((b) => b.id === c.bigTaskId);
          const board = BOARDS.find((b) => b.id === bt?.boardId);
          return (
            <div className="queue-row" key={c.id}>
              <Avatar id={c.pic} size={28} />
              <div className="queue-main" onClick={() => onOpenTask(c)}>
                <div className="queue-title">{c.title}</div>
                <div className="muted small">
                  {board?.name} / {bt?.name}
                </div>
              </div>
              <span className="mono">{c.actualPct}%</span>
              <button
                className="approve-btn"
                onClick={() => setReviewedState((s) => ({ ...s, [c.id]: true }))}
              >
                <Check size={13} aria-hidden="true" /> Sudah gue lihat
              </button>
            </div>
          );
        })}
        {unreviewed.length === 0 && (
          <div className="empty-note">Semua task udah lo review. Rapi.</div>
        )}
      </div>
    </div>
  );
}

function TaskDetail({ card, onClose, reviewedState, setReviewedState }) {
  if (!card) return null;
  const bt = BIG_TASKS.find((b) => b.id === card.bigTaskId);
  const board = BOARDS.find((b) => b.id === bt?.boardId);
  const person = TEAM.find((t) => t.id === card.pic);
  const daysLeft = daysBetween(TODAY, new Date(card.deadline));
  const isReviewed = reviewedState[card.id];

  return (
    <div className="overlay" onClick={onClose}>
      <div className="panel" onClick={(e) => e.stopPropagation()}>
        <div className="panel-header">
          <span className="titlebar-title" style={{ fontSize: 11 }}>
            {board?.name} / {bt?.name}
          </span>
          <button className="icon-btn" onClick={onClose} aria-label="Tutup">
            <X size={12} aria-hidden="true" />
          </button>
        </div>
        <h3 className="panel-title">{card.title}</h3>

        <div className="panel-row">
          <Avatar id={card.pic} size={28} />
          <div>
            <div className="panel-row-name">{person?.name}</div>
            <div className="muted small">{person?.role}</div>
          </div>
        </div>

        <div className="panel-grid">
          <div className="panel-field">
            <div className="muted small">Progress</div>
            <div className="mono panel-big">{card.actualPct}%</div>
          </div>
          <div className="panel-field">
            <div className="muted small">Deadline komitmen</div>
            <div className="mono panel-big">{card.deadline}</div>
            <div className={`small ${daysLeft < 0 ? "days-late" : "muted"}`}>
              {daysLeft < 0 ? `telat ${Math.abs(daysLeft)} hari` : `${daysLeft} hari lagi`}
            </div>
          </div>
        </div>

        <div className="panel-field">
          <div className="muted small">Lampiran ({card.attachments})</div>
          {card.attachments > 0 ? (
            <div className="attach-grid">
              {Array.from({ length: card.attachments }).map((_, i) => (
                <div className="attach-thumb" key={i}>
                  <File size={16} aria-hidden="true" />
                </div>
              ))}
            </div>
          ) : (
            <div className="empty-note">Belum ada lampiran.</div>
          )}
        </div>

        <div className="panel-field review-toggle-row">
          <div>
            <div className="panel-row-name">Ditinjau SPV</div>
            <div className="muted small">
              {isReviewed ? "Sudah dilihat Lugina" : "Belum ditinjau"}
            </div>
          </div>
          <button
            className={`toggle ${isReviewed ? "toggle-on" : ""}`}
            onClick={() =>
              setReviewedState((s) => ({ ...s, [card.id]: !s[card.id] }))
            }
            aria-label="Tandai sudah ditinjau"
          >
            <span className="toggle-dot" />
          </button>
        </div>
      </div>
    </div>
  );
}

function ProfileModal({ onClose, name, setName, initials, setInitials }) {
  const [pw1, setPw1] = useState("");
  const [pw2, setPw2] = useState("");
  return (
    <div className="overlay" onClick={onClose}>
      <div className="modal-box" onClick={(e) => e.stopPropagation()}>
        <div className="panel-header">
          <span className="titlebar-title" style={{ fontSize: 11 }}>My Profile</span>
          <button className="icon-btn" onClick={onClose} aria-label="Tutup"><X size={12} aria-hidden="true" /></button>
        </div>
        <div className="modal-body">
          <div className="panel-field">
            <div className="muted small">Nama</div>
            <input className="inline-input" value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <div className="panel-field">
            <div className="muted small">Inisial avatar</div>
            <input className="inline-input" maxLength={2} value={initials} onChange={(e) => setInitials(e.target.value.toUpperCase())} />
          </div>
          <div className="panel-field">
            <div className="muted small">Password baru</div>
            <input type="password" className="inline-input" value={pw1} onChange={(e) => setPw1(e.target.value)} placeholder="••••••••" />
          </div>
          <div className="panel-field">
            <div className="muted small">Konfirmasi password</div>
            <input type="password" className="inline-input" value={pw2} onChange={(e) => setPw2(e.target.value)} placeholder="••••••••" />
          </div>
          <button className="quick-btn quick-btn-done" onClick={onClose}>
            <Check size={11} aria-hidden="true" /> Simpan perubahan
          </button>
        </div>
      </div>
    </div>
  );
}

function SettingsModal({ onClose, settingsTab, setSettingsTab, theme, setTheme }) {
  return (
    <div className="overlay" onClick={onClose}>
      <div className="modal-box modal-box-wide" onClick={(e) => e.stopPropagation()}>
        <div className="panel-header">
          <span className="titlebar-title" style={{ fontSize: 11 }}>Settings</span>
          <button className="icon-btn" onClick={onClose} aria-label="Tutup"><X size={12} aria-hidden="true" /></button>
        </div>
        <div className="settings-tabs">
          <button className={`role-filter-pill ${settingsTab === "users" ? "role-filter-pill-active" : ""}`} onClick={() => setSettingsTab("users")}>
            Manajemen user
          </button>
          <button className={`role-filter-pill ${settingsTab === "theme" ? "role-filter-pill-active" : ""}`} onClick={() => setSettingsTab("theme")}>
            Tema aplikasi
          </button>
        </div>
        <div className="modal-body">
          {settingsTab === "users" && (
            <table className="sheet-table">
              <thead>
                <tr><th>Nama</th><th>Role</th><th>Tim</th></tr>
              </thead>
              <tbody>
                {TEAM.map((m) => (
                  <tr key={m.id}>
                    <td><Avatar id={m.id} size={18} /> <span style={{ marginLeft: 6 }}>{m.name}</span></td>
                    <td className="muted small">{m.role}</td>
                    <td className="muted small">{m.team}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
          {settingsTab === "theme" && (
            <div className="theme-grid">
              {Object.entries(THEMES).map(([key, val]) => (
                <button
                  key={key}
                  className={`theme-option ${theme === key ? "theme-option-active" : ""}`}
                  onClick={() => setTheme(key)}
                >
                  <span className="theme-swatch-row">
                    <span className="theme-swatch" style={{ background: val.titlebarA }} />
                    <span className="theme-swatch" style={{ background: val.face }} />
                    <span className="theme-swatch" style={{ background: val.winBlue }} />
                  </span>
                  <span className="small">{val.label}</span>
                </button>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default function RndOpsApp() {
  const [tab, setTab] = useState("dashboard");
  const [activeBoard, setActiveBoard] = useState(BOARDS[0].id);
  const [openTask, setOpenTask] = useState(null);
  const [cards, setCards] = useState(TASK_CARDS);
  const initialReviewed = useMemo(() => {
    const o = {};
    TASK_CARDS.forEach((c) => (o[c.id] = c.reviewed));
    return o;
  }, []);
  const [reviewedState, setReviewedState] = useState(initialReviewed);
  const pendingReview = cards.filter((c) => !reviewedState[c.id]).length;

  const bumpProgress = (id, delta) => {
    setCards((prev) =>
      prev.map((c) => (c.id === id ? { ...c, actualPct: Math.min(100, c.actualPct + delta) } : c))
    );
  };
  const markDone = (id) => {
    setCards((prev) => prev.map((c) => (c.id === id ? { ...c, actualPct: 100 } : c)));
  };

  const [dailyTasks, setDailyTasks] = useState(DAILY_TASKS_INITIAL);
  const [extraBigTasks, setExtraBigTasks] = useState([]);

  const addDailyTask = (bigTaskId, title, start, end, pic) => {
    const startD = new Date(start);
    const endD = new Date(end);
    const days = [];
    for (let d = new Date(startD); d <= endD; d.setDate(d.getDate() + 1)) {
      days.push({ date: d.toISOString().slice(0, 10), planned: "", done: false, blocker: "", note: "" });
    }
    const bt = [...BIG_TASKS, ...extraBigTasks].find((b) => b.id === bigTaskId);
    setDailyTasks((prev) => [
      ...prev,
      { id: `dt${Date.now()}`, bigTaskId, title, pic: pic || bt?.pic || "lugina", startDate: start, endDate: end, days },
    ]);
  };

  const updateDayField = (dailyTaskId, date, field, value) => {
    setDailyTasks((prev) =>
      prev.map((dt) =>
        dt.id === dailyTaskId
          ? { ...dt, days: dt.days.map((d) => (d.date === date ? { ...d, [field]: value } : d)) }
          : dt
      )
    );
  };

  const toggleDayDone = (dailyTaskId, date) => {
    setDailyTasks((prev) =>
      prev.map((dt) =>
        dt.id === dailyTaskId
          ? {
              ...dt,
              days: dt.days.map((d) =>
                d.date === date ? { ...d, done: !d.done, blocker: !d.done ? "" : d.blocker } : d
              ),
            }
          : dt
      )
    );
  };

  const addBigTask = (boardId, name) => {
    const today = TODAY.toISOString().slice(0, 10);
    const later = new Date(TODAY);
    later.setDate(later.getDate() + 14);
    setExtraBigTasks((prev) => [
      ...prev,
      { id: `bt${Date.now()}`, boardId, name, startDate: today, deadline: later.toISOString().slice(0, 10), pic: "lugina" },
    ]);
  };

  const [comments, setComments] = useState(COMMENTS_INITIAL);
  const addComment = (bigTaskId, dailyTaskId, author, text) => {
    setComments((prev) => [
      ...prev,
      { id: `cm${Date.now()}`, bigTaskId, dailyTaskId, author, text, date: TODAY.toISOString().slice(0, 10) },
    ]);
  };

  const [cheatSheet, setCheatSheet] = useState(CHEAT_SHEET_INITIAL);
  const addCheatSheetItem = (boardId, type, title, value) => {
    setCheatSheet((prev) => [
      ...prev,
      { id: `cs${Date.now()}`, boardId, type, title, value, author: CURRENT_USER, date: TODAY.toISOString().slice(0, 10) },
    ]);
  };

  const [signedBigTasks, setSignedBigTasks] = useState({});
  const signBigTask = (bigTaskId) => {
    setSignedBigTasks((prev) => ({
      ...prev,
      [bigTaskId]: { signedBy: CURRENT_USER, date: TODAY.toISOString().slice(0, 10) },
    }));
  };
  const unsignBigTask = (bigTaskId) => {
    setSignedBigTasks((prev) => {
      const next = { ...prev };
      delete next[bigTaskId];
      return next;
    });
  };

  const [weekStart, setWeekStart] = useState(getWeekStart(TODAY));
  const [weeklyPushState, setWeeklyPushState] = useState({});
  const pushWeekly = (bigTaskId, weekStartStr) => {
    const key = `${bigTaskId}_${weekStartStr}`;
    setWeeklyPushState((prev) => ({
      ...prev,
      [key]: {
        callbackId: prev[key]?.callbackId || `hr-${bigTaskId}-${weekStartStr}`,
        pushedAt: TODAY.toISOString().slice(0, 10),
      },
    }));
  };

  const [theme, setTheme] = useState("retro-light");
  const t = THEMES[theme];
  const [showUserMenu, setShowUserMenu] = useState(false);
  const [showNotif, setShowNotif] = useState(false);
  const [showProfile, setShowProfile] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [settingsTab, setSettingsTab] = useState("users");
  const [signedOut, setSignedOut] = useState(false);
  const [profileName, setProfileName] = useState("Lugina");
  const [profileInitials, setProfileInitials] = useState("LG");

  if (signedOut) {
    return (
      <div className={`app signedout-screen ${t.modern ? "theme-modern" : ""}`}>
        <style>{`
          .app { --face: ${t.face}; --content-bg: ${t.contentBg}; --text-primary: ${t.textPrimary};
            --text-muted: ${t.textMuted}; --win-blue: ${t.winBlue}; background: var(--face); color: var(--text-primary);
            font-family: Tahoma, sans-serif; border: 2px outset ${t.face}; padding: 40px 20px; text-align: center; }
          .signedout-btn { background: var(--win-blue); color: #fff; border: none; padding: 8px 16px; cursor: pointer; margin-top: 12px; font-family: inherit; }
        `}</style>
        <p>Anda telah keluar dari R&amp;D Ops.</p>
        <button className="signedout-btn" onClick={() => setSignedOut(false)}>Masuk kembali</button>
      </div>
    );
  }

  return (
    <div className={`app ${t.modern ? "theme-modern" : ""}`}>
      <style>{`
        .app {
          --face: ${t.face};
          --face-light: #FFFFFF;
          --content-bg: ${t.contentBg};
          --content-alt: ${t.contentAlt};
          --text-primary: ${t.textPrimary};
          --text-muted: ${t.textMuted};
          --titlebar-a: ${t.titlebarA};
          --titlebar-b: ${t.titlebarB};
          --win-blue: ${t.winBlue};
          --win-blue-light: ${t.winBlueLight};
          --win-green: ${t.winGreen};
          --win-red: ${t.winRed};
          --win-amber: ${t.winAmber};
          background: var(--face);
          color: var(--text-primary);
          font-family: Tahoma, 'MS Sans Serif', 'Segoe UI', Arial, sans-serif;
          font-size: 11px;
          border: 2px outset ${t.face};
          padding: 0;
          overflow: hidden;
        }
        .mono { font-family: 'Courier New', monospace; }
        .muted { color: var(--text-muted); }
        .small { font-size: 10.5px; }
        .accent-text { color: var(--win-blue); font-weight: bold; }
        .dot-sep { color: var(--text-muted); margin: 0 2px; }

        .titlebar {
          display: flex; align-items: center; justify-content: space-between;
          padding: 3px 3px 3px 6px;
          background: linear-gradient(90deg, var(--titlebar-a), var(--titlebar-b));
          color: #FFFFFF;
        }
        .titlebar-left { display: flex; align-items: center; gap: 6px; }
        .titlebar-icon {
          width: 15px; height: 15px; background: var(--face); border: 1px outset var(--face);
          display: flex; align-items: center; justify-content: center; font-size: 10px; font-weight: bold; color: var(--win-blue);
        }
        .titlebar-title { font-size: 11px; font-weight: bold; letter-spacing: 0.2px; }
        .titlebar-btns { display: flex; gap: 2px; }
        .titlebar-btn {
          width: 16px; height: 14px; background: var(--face); border: 1px outset var(--face);
          display: flex; align-items: center; justify-content: center; font-size: 9px; font-weight: bold; color: #000; line-height: 1;
          border-radius: 0;
        }

        .menubar {
          display: flex; gap: 14px; padding: 3px 8px; background: var(--face);
          border-bottom: 1px solid #C3C8CC; font-size: 11px;
        }
        .menubar span { cursor: default; }

        .topbar {
          display: flex; align-items: center; justify-content: space-between;
          padding: 4px 6px; background: var(--face); border-bottom: 1px solid #C3C8CC;
          gap: 8px;
        }
        .brand-block { display: none; }
        .brand-mark { display: none; }
        .brand-name { display: none; }
        .brand-sub { display: none; }
        .tabs { display: flex; gap: 0; }
        .tab-btn {
          background: var(--face); border: 1px solid #C3C8CC; border-bottom: none; color: var(--text-primary);
          font-size: 11px; font-weight: normal; padding: 5px 12px; margin-right: -1px;
          cursor: pointer; font-family: inherit; display: flex; align-items: center; gap: 5px;
          position: relative; top: 1px;
        }
        .tab-btn:hover { background: var(--face-light); }
        .tab-btn-active {
          background: var(--content-bg); border-bottom: 1px solid var(--content-bg); font-weight: bold; z-index: 1;
        }
        .review-badge {
          background: var(--win-red); color: #FFFFFF; font-size: 10px; font-weight: bold;
          border: 1px outset var(--win-red); padding: 0px 5px; margin-left: 2px; font-family: 'Courier New', monospace;
        }
        .topbar-right { display: flex; align-items: center; gap: 10px; padding: 4px 6px; }

        .content { padding: 12px; min-height: 420px; background: var(--content-bg); border: 1px solid #C3C8CC; border-top: none; }

        .dash-heading { display: flex; flex-direction: column; gap: 2px; margin-bottom: 12px; }
        .dash-heading-title { font-size: 12px; font-weight: bold; }

        .stat-row { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; margin-bottom: 16px; }
        .stat-row-8 { grid-template-columns: repeat(8, 1fr); }
        .stat-card { background: var(--face); border: 2px groove var(--face); padding: 7px 8px; }
        .stat-label { font-size: 10px; color: var(--text-muted); margin-bottom: 5px; line-height: 1.2; }
        .stat-value {
          font-size: 15px; font-weight: bold; font-family: 'Courier New', monospace;
          background: var(--content-bg); border: 1px inset var(--face); padding: 2px 5px; display: inline-block;
        }
        .tone-warn .stat-value { color: var(--win-red); }
        .tone-good .stat-value { color: var(--win-green); }
        .tone-accent .stat-value { color: var(--win-blue); }

        .two-col { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin-bottom: 18px; }
        .two-col .section { margin-bottom: 0; }

        .chart-legend { display: flex; flex-direction: column; gap: 6px; flex: 1; }
        .chart-legend-row { display: flex; align-items: center; gap: 6px; }
        .chart-legend-swatch { width: 9px; height: 9px; flex-shrink: 0; border: 1px solid #808080; }

        .breakdown-list { display: flex; flex-direction: column; gap: 8px; }
        .breakdown-row { display: grid; grid-template-columns: 100px 1fr 24px; align-items: center; gap: 8px; }
        .breakdown-label { font-size: 11px; color: var(--text-muted); }
        .breakdown-track { height: 13px; background: var(--content-bg); border: 1px inset var(--face); overflow: hidden; }
        .breakdown-fill { height: 100%; background-image: repeating-linear-gradient(90deg, currentColor 0px, currentColor 6px, transparent 6px, transparent 8px); }
        .fill-neutral { background: var(--text-muted); color: var(--text-muted); }
        .fill-brass { background: var(--win-blue); color: var(--win-blue); }
        .breakdown-value { font-size: 11px; text-align: right; font-family: 'Courier New', monospace; }

        .sheet-table { width: 100%; border-collapse: collapse; font-size: 11px; }
        .sheet-table th {
          text-align: left; color: #FFFFFF; font-weight: bold; font-size: 10.5px;
          padding: 4px 7px; background: var(--win-blue);
        }
        .sheet-table td { padding: 5px 7px; border-bottom: 1px solid #D6DADD; }
        .sheet-table tr:nth-child(even) td { background: var(--content-alt); }

        .section { margin-bottom: 18px; }
        .section-title {
          font-size: 11px; font-weight: bold; margin-bottom: 8px; padding: 3px 6px;
          background: linear-gradient(90deg, var(--titlebar-a), var(--titlebar-b)); color: #FFFFFF;
        }

        .attention-list { display: flex; flex-direction: column; gap: 4px; }
        .attention-row {
          display: grid; grid-template-columns: 1.4fr 1.3fr auto auto; align-items: center; gap: 12px;
          background: var(--face); border: 1px solid #C3C8CC; padding: 6px 10px;
        }
        .attention-main { display: flex; flex-direction: column; gap: 1px; }
        .attention-name { font-size: 11px; font-weight: bold; }
        .attention-bar { min-width: 140px; }
        .days-late { color: var(--win-red); font-weight: bold; }

        .empty-note { color: var(--text-muted); font-size: 11px; padding: 8px 2px; }

        .team-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; }
        .team-card { background: var(--face); border: 2px groove var(--face); padding: 8px; }
        .team-card-top { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
        .team-name { font-size: 11px; font-weight: bold; }
        .team-card-stats { display: flex; flex-direction: column; gap: 3px; font-size: 10.5px; }

        .avatar {
          border: 1px outset var(--face); background: var(--content-bg); color: var(--win-blue);
          border-radius: 0;
          display: flex; align-items: center; justify-content: center;
          font-family: 'Courier New', monospace; font-weight: bold;
          flex-shrink: 0;
        }

        .dualbar { display: flex; flex-direction: column; gap: 3px; }
        .dualbar-track { position: relative; height: 12px; background: var(--content-bg); border: 1px inset var(--face); overflow: visible; }
        .dualbar-fill { position: absolute; top: 0; left: 0; height: 100%; background-image: repeating-linear-gradient(90deg, currentColor 0px, currentColor 6px, transparent 6px, transparent 8px); }
        .fill-good { background: var(--win-green); color: var(--win-green); }
        .fill-bad { background: var(--win-amber); color: var(--win-amber); }
        .dualbar-tick { position: absolute; top: -2px; width: 2px; height: 16px; background: var(--win-red); }
        .dualbar-legend { display: flex; justify-content: space-between; font-size: 10px; font-family: 'Courier New', monospace; }
        .dualbar-legend-muted { color: var(--text-muted); }

        .badge {
          font-size: 10px; font-weight: bold; padding: 1px 6px 1px 5px;
          white-space: nowrap; font-family: inherit;
          border: 1px outset var(--face); background: var(--face); color: var(--text-primary);
          border-radius: 0;
          display: inline-flex; align-items: center; gap: 4px;
        }
        .badge::before { content: ""; width: 6px; height: 6px; display: inline-block; }
        .badge-win::before { background: var(--win-green); }
        .badge-lose::before { background: var(--win-red); }
        .badge-ontrack::before { background: var(--win-green); }
        .badge-offtrack::before { background: var(--win-amber); }
        .badge-onprogress::before { background: var(--win-blue); }
        .badge-onprogress { color: var(--win-blue); border-color: var(--win-blue); }

        .board-pills { display: flex; gap: 0; margin-bottom: 0; flex-wrap: wrap; }
        .board-pill {
          background: var(--face); border: 1px solid #C3C8CC; border-bottom: none; color: var(--text-primary);
          font-size: 11px; font-weight: normal; padding: 6px 14px; cursor: pointer; margin-right: -1px;
          font-family: inherit; position: relative; top: 1px;
        }
        .board-pill-active { background: var(--content-bg); font-weight: bold; border-bottom: 1px solid var(--content-bg); z-index: 1; }
        .board-pill-ghost { border-style: dashed; display: flex; align-items: center; gap: 4px; }

        .kanban-scroll { display: flex; gap: 10px; overflow-x: auto; padding: 12px 0 6px; border-top: 1px solid #C3C8CC; }
        .column { background: var(--face); border: 2px groove var(--face); width: 250px; flex-shrink: 0; display: flex; flex-direction: column; }
        .column-ghost { align-items: center; justify-content: center; min-height: 140px; }
        .column-header { padding: 10px 10px 8px; border-bottom: 1px solid #C3C8CC; display: flex; flex-direction: column; gap: 7px; }
        .column-header-top { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
        .column-title { font-size: 11px; font-weight: bold; }
        .column-header-meta { display: flex; align-items: center; gap: 6px; font-size: 10.5px; }
        .column-body { padding: 8px; display: flex; flex-direction: column; gap: 6px; background: var(--content-bg); margin: 1px; }

        .task-card {
          background: var(--content-bg); border: 1px solid #B7BCC0;
          border-radius: 0;
          padding: 8px; text-align: left; cursor: pointer; display: flex; flex-direction: column; gap: 6px;
          color: var(--text-primary); font-family: inherit; width: 100%;
        }
        .task-card:hover { background: var(--win-blue); color: #FFFFFF; border-color: var(--win-blue); }
        .task-card:hover .muted, .task-card:hover .task-card-bottom { color: #DCE6F5; }
        .task-card-top { display: flex; align-items: flex-start; justify-content: space-between; gap: 6px; }
        .task-title { font-size: 11px; line-height: 1.35; }
        .review-dot { width: 6px; height: 6px; background: var(--win-red); flex-shrink: 0; margin-top: 4px; }
        .task-card-bottom { display: flex; align-items: center; gap: 7px; font-size: 10px; color: var(--text-muted); font-family: 'Courier New', monospace; }
        .task-pct { color: inherit; }
        .task-attach { display: flex; align-items: center; gap: 2px; margin-left: auto; }

        .add-card-ghost {
          background: var(--face); border: 1px outset var(--face); color: var(--text-primary);
          border-radius: 0;
          padding: 7px; font-size: 11px; cursor: pointer; display: flex; align-items: center;
          justify-content: center; gap: 4px; font-family: inherit;
        }
        .add-card-ghost:active { border-style: inset; }
        .add-column-ghost { width: 100%; margin: 20px; }

        .queue-list { display: flex; flex-direction: column; gap: 4px; }
        .queue-row {
          display: flex; align-items: center; gap: 10px; background: var(--face);
          border: 1px solid #C3C8CC; padding: 7px 10px;
        }
        .queue-main { flex: 1; cursor: pointer; }
        .queue-title { font-size: 11px; margin-bottom: 2px; }
        .approve-btn {
          background: var(--face); border: 1px outset var(--face); color: var(--text-primary);
          border-radius: 0;
          font-size: 11px; padding: 5px 10px; cursor: pointer; display: flex;
          align-items: center; gap: 4px; font-family: inherit;
        }
        .approve-btn:active { border-style: inset; }

        .overlay {
          position: absolute; inset: 0; background: rgba(0,0,0,0.35);
          display: flex; justify-content: flex-end; z-index: 10;
        }
        .panel {
          width: 340px; max-width: 88%; background: var(--face); border: 2px outset var(--face); border-right: none;
          padding: 0; overflow-y: auto; display: flex; flex-direction: column; gap: 0;
        }
        .panel-header {
          display: flex; align-items: center; justify-content: space-between;
          background: linear-gradient(90deg, var(--titlebar-a), var(--titlebar-b)); color: #FFFFFF; padding: 5px 8px;
        }
        .icon-btn { background: var(--face); border: 1px outset var(--face); color: #000; cursor: pointer; width: 18px; height: 16px; border-radius: 0; display: flex; align-items: center; justify-content: center; }
        .icon-btn:active { border-style: inset; }
        .icon-btn:hover { color: #000; }
        .panel-title { font-size: 12px; font-weight: bold; margin: 0; padding: 12px 14px 4px; }
        .panel-row { display: flex; align-items: center; gap: 10px; padding: 0 14px; }
        .panel-row-name { font-size: 11px; font-weight: bold; }
        .panel-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; padding: 10px 14px 0; }
        .panel-field { display: flex; flex-direction: column; gap: 4px; padding: 10px 14px 0; }
        .panel-grid .panel-field { padding: 0; }
        .panel-big {
          font-size: 14px; background: var(--content-bg); border: 1px inset var(--face); padding: 3px 6px; display: inline-block;
        }
        .attach-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 5px; }
        .attach-thumb {
          aspect-ratio: 1; background: var(--content-bg); border: 1px inset var(--face);
          display: flex; align-items: center; justify-content: center; color: var(--text-muted);
        }
        .review-toggle-row { flex-direction: row; align-items: center; justify-content: space-between; padding: 10px 14px 14px; }
        .toggle {
          width: 40px; height: 18px; background: var(--content-bg); border: 1px inset var(--face);
          position: relative; cursor: pointer; flex-shrink: 0;
        }
        .toggle-dot { position: absolute; top: 1px; left: 1px; width: 15px; height: 15px; background: var(--face); border: 1px outset var(--face); transition: transform 0.1s; }
        .toggle-on { background: var(--win-blue-light); }
        .toggle-on .toggle-dot { transform: translateX(21px); }

        .task-quick-actions { display: flex; gap: 5px; margin-top: 2px; }
        .quick-btn {
          background: var(--face); border: 1px outset var(--face); color: var(--text-primary);
          border-radius: 0; font-size: 10px; padding: 3px 7px; cursor: pointer; font-family: inherit;
          display: flex; align-items: center; gap: 3px; flex: 1; justify-content: center;
        }
        .quick-btn:active { border-style: inset; }
        .quick-btn-done { color: var(--win-green); font-weight: bold; }

        .board-pills-row { display: flex; align-items: flex-end; justify-content: space-between; gap: 8px; }
        .view-toggle { display: flex; gap: 0; margin-bottom: 2px; }
        .view-toggle-btn {
          background: var(--face); border: 1px outset var(--face); color: var(--text-primary);
          border-radius: 0; font-size: 10.5px; padding: 5px 10px; cursor: pointer; font-family: inherit;
          display: flex; align-items: center; gap: 4px;
        }
        .view-toggle-btn:active, .view-toggle-btn-active { border-style: inset; background: var(--content-bg); font-weight: bold; }

        .list-view-table { margin-top: 10px; }
        .list-title-cell { cursor: pointer; }
        .list-title-cell:hover { text-decoration: underline; }
        .list-progress-cell { display: flex; align-items: center; gap: 6px; }
        .list-actions { display: flex; gap: 4px; }

        .project-layout { display: grid; grid-template-columns: 200px 1fr; gap: 12px; margin-top: 14px; }
        .bigtask-list { display: flex; flex-direction: column; gap: 6px; }
        .bigtask-list-item {
          display: flex; flex-direction: column; align-items: flex-start; gap: 2px;
          background: var(--face); border: 1px solid #C3C8CC; padding: 7px 9px; cursor: pointer;
          font-family: inherit; width: 100%;
        }
        .bigtask-list-item .dualbar-track { width: 100%; height: 5px; }
        .bigtask-list-item-active { background: var(--content-bg); border-color: var(--win-blue); border-width: 2px; }
        .bigtask-list-top { display: flex; align-items: center; justify-content: space-between; gap: 6px; width: 100%; }
        .bigtask-list-bottom { display: flex; align-items: center; justify-content: space-between; width: 100%; margin-top: 2px; }
        .bigtask-list-name { font-size: 11px; font-weight: bold; text-align: left; }
        .bigtask-detail-title { display: flex; align-items: center; justify-content: space-between; }

        .bigtask-detail { background: var(--content-bg); border: 1px solid #C3C8CC; padding: 10px; min-height: 200px; }

        .daily-task-card { border: 1px solid #C3C8CC; margin-bottom: 10px; }
        .daily-task-head {
          display: flex; align-items: center; justify-content: space-between; gap: 8px;
          background: var(--face); padding: 6px 9px; border-bottom: 1px solid #C3C8CC;
        }
        .daily-task-title { font-size: 11px; font-weight: bold; }
        .daily-task-head-right { display: flex; align-items: center; gap: 8px; }

        .daily-day-table { font-size: 10.5px; }
        .daily-day-table th { padding: 4px 6px; }
        .daily-day-table td { padding: 4px 6px; vertical-align: middle; }
        .row-weekend { background: var(--content-alt); }
        .lembur-badge {
          display: inline-block; margin-left: 5px; font-size: 9px; font-weight: bold; color: #FFFFFF;
          background: var(--win-red); padding: 1px 4px; vertical-align: middle;
        }

        .day-status-btn {
          border: 1px outset var(--face); border-radius: 0; font-size: 10px; padding: 3px 8px; cursor: pointer;
          font-family: inherit;
        }
        .day-status-btn:active { border-style: inset; }
        .day-status-done { background: var(--win-blue-light); color: var(--win-green); font-weight: bold; }
        .day-status-open { background: var(--face); color: var(--win-red); }

        .inline-input {
          border: 1px inset var(--face); background: var(--content-bg); color: var(--text-primary);
          font-family: inherit; font-size: 11px; padding: 4px 6px; width: 100%; box-sizing: border-box;
        }
        .inline-input-cell { font-size: 10.5px; padding: 3px 5px; }
        .inline-form {
          display: flex; align-items: center; gap: 6px; background: var(--face);
          border: 1px solid #C3C8CC; padding: 8px; margin-top: 4px;
        }
        .inline-form-daily { flex-direction: column; align-items: stretch; }
        .inline-form-dates { display: flex; align-items: center; gap: 6px; }
        .inline-form-row { display: flex; align-items: center; gap: 8px; }
        .inline-form-actions { display: flex; gap: 6px; }
        .role-filter-pills { display: flex; gap: 3px; flex-wrap: wrap; }
        .role-filter-pill {
          background: var(--content-bg); border: 1px solid #C3C8CC; border-radius: 0;
          font-size: 10px; padding: 2px 7px; cursor: pointer; font-family: inherit;
        }
        .role-filter-pill-active { background: var(--win-blue); color: #FFFFFF; border-color: var(--win-blue); font-weight: bold; }
        .team-tag {
          font-size: 9px; font-weight: bold; color: #FFFFFF; background: var(--win-amber);
          padding: 1px 5px;
        }
        .comment-jump-btn {
          background: var(--content-bg); border: 1px outset var(--face); color: var(--win-blue);
          border-radius: 0; font-size: 10px; padding: 3px 7px; cursor: pointer; font-family: inherit;
          display: flex; align-items: center; gap: 4px;
        }
        .comment-jump-btn:active { border-style: inset; }

        .review-clone-group { display: flex; align-items: center; gap: 4px; }
        .review-clone-btn {
          background: var(--content-bg); border: 1px outset var(--face); color: var(--win-blue);
          border-radius: 0; font-size: 10px; padding: 3px 7px; cursor: pointer; font-family: inherit; font-weight: bold;
        }
        .review-clone-btn:active { border-style: inset; }

        .project-status-banner {
          display: flex; align-items: center; gap: 8px; background: var(--face);
          border: 1px solid #C3C8CC; padding: 6px 10px; margin-bottom: 10px;
        }
        .project-status-dot { width: 8px; height: 8px; border-radius: 50%; }
        .status-done { background: var(--win-green); }
        .status-progress { background: var(--win-blue); }

        .bigtask-detail-title-right { display: flex; align-items: center; gap: 8px; }
        .sign-btn {
          background: var(--content-bg); border: 1px outset var(--face); color: var(--text-primary);
          border-radius: 0; font-size: 10.5px; padding: 4px 9px; cursor: pointer; font-family: inherit;
        }
        .sign-btn:active:not(:disabled) { border-style: inset; }
        .sign-btn:disabled { color: var(--text-muted); cursor: not-allowed; opacity: 0.6; }
        .sign-btn-active {
          background: var(--win-blue-light); color: var(--win-green); font-weight: bold; display: flex;
          align-items: center; gap: 4px;
        }

        .weekplan-header { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; }
        .weekplan-range { font-family: 'Courier New', monospace; font-size: 12px; font-weight: bold; }
        .weekplan-table td, .weekplan-table th { white-space: nowrap; }

        .icon-only-btn {
          position: relative; background: transparent; border: none; color: var(--text-primary);
          cursor: pointer; padding: 4px; display: flex; align-items: center;
        }
        .notif-wrap, .user-menu-wrap { position: relative; }
        .notif-badge {
          position: absolute; top: -2px; right: -2px; background: var(--win-red); color: #fff;
          font-size: 8px; font-weight: bold; border-radius: 999px; padding: 1px 4px; min-width: 12px; text-align: center;
        }
        .user-menu-trigger { background: transparent; border: none; cursor: pointer; padding: 0; }
        .dropdown-panel {
          position: absolute; top: calc(100% + 4px); right: 0; z-index: 30; width: 220px;
          background: var(--content-bg); border: 2px outset var(--face); padding: 8px;
        }
        .dropdown-title { font-size: 10.5px; font-weight: bold; margin-bottom: 6px; }
        .notif-item { display: flex; align-items: center; gap: 6px; padding: 4px 2px; }
        .user-menu-head { display: flex; align-items: center; gap: 8px; padding: 2px 2px 8px; }
        .dropdown-divider { border-top: 1px solid #C3C8CC; margin: 4px 0; }
        .dropdown-item {
          display: flex; align-items: center; gap: 8px; width: 100%; background: transparent; border: none;
          padding: 6px 4px; cursor: pointer; font-family: inherit; font-size: 11px; color: var(--text-primary);
          text-align: left;
        }
        .dropdown-item:hover { background: var(--win-blue-light); }
        .dropdown-item-danger { color: var(--win-red); }

        .modal-box {
          width: 320px; max-width: 90%; background: var(--face); border: 2px outset var(--face);
          margin: auto;
        }
        .modal-box-wide { width: 460px; }
        .modal-body { padding: 14px; display: flex; flex-direction: column; gap: 10px; background: var(--content-bg); }
        .settings-tabs { display: flex; gap: 4px; padding: 8px 14px 0; background: var(--content-bg); }

        .theme-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 10px; }
        .theme-option {
          display: flex; flex-direction: column; gap: 6px; background: var(--face); border: 1px solid #C3C8CC;
          padding: 10px; cursor: pointer; font-family: inherit;
        }
        .theme-option-active { border: 2px solid var(--win-blue); }
        .theme-swatch-row { display: flex; gap: 4px; }
        .theme-swatch { width: 20px; height: 20px; border: 1px solid rgba(0,0,0,0.2); }

        .theme-modern .app,
        .theme-modern.app { border-radius: 10px; }
        .theme-modern .titlebar { border-bottom: 1px solid #0000001a; }
        .theme-modern .task-card, .theme-modern .stat-card, .theme-modern .team-card,
        .theme-modern .daily-task-card, .theme-modern .cheatsheet-row, .theme-modern .comment-body,
        .theme-modern .modal-box, .theme-modern .dropdown-panel, .theme-modern .column,
        .theme-modern .queue-row, .theme-modern .attention-row {
          border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.12); border-color: transparent;
        }
        .theme-modern .quick-btn, .theme-modern .approve-btn, .theme-modern .add-card-ghost,
        .theme-modern .board-pill, .theme-modern .tab-btn, .theme-modern .role-filter-pill,
        .theme-modern .sign-btn, .theme-modern .day-status-btn, .theme-modern .comment-jump-btn,
        .theme-modern .review-clone-btn, .theme-modern .icon-btn {
          border-radius: 6px; border-style: solid !important;
        }
        .theme-modern .badge { border-radius: 999px; }
        .theme-modern .titlebar-icon, .theme-modern .brand-mark { border-radius: 6px; }

        .comment-section { margin-top: 16px; border-top: 2px solid var(--win-blue); padding-top: 10px; }
        .comment-filter-row { display: flex; gap: 4px; flex-wrap: wrap; margin-bottom: 10px; }
        .comment-list { display: flex; flex-direction: column; gap: 8px; margin-bottom: 10px; }
        .comment-row { display: flex; gap: 8px; }
        .comment-body { flex: 1; background: var(--face); border: 1px solid #C3C8CC; padding: 6px 9px; }
        .comment-meta { display: flex; align-items: center; gap: 6px; margin-bottom: 3px; }
        .comment-author { font-size: 11px; font-weight: bold; }
        .comment-scope-tag {
          font-size: 9px; font-weight: bold; color: var(--win-blue); border: 1px solid var(--win-blue);
          padding: 0px 5px;
        }
        .comment-text { font-size: 11px; line-height: 1.4; }
        .comment-composer { display: flex; flex-direction: column; gap: 6px; background: var(--content-alt); border: 1px solid #C3C8CC; padding: 8px; }
        .comment-composer-row { display: flex; gap: 6px; }
        .comment-composer-row .inline-input:first-child { flex-shrink: 0; }
        .comment-composer-row select.inline-input { flex: 1; }
        .comment-input-wrap { align-items: flex-start; }
        .comment-input-relative { position: relative; flex: 1; }
        .mention-tag { color: var(--win-blue); font-weight: bold; }
        .mention-popup {
          position: absolute; top: calc(100% + 2px); left: 0; right: 0; z-index: 20;
          background: var(--content-bg); border: 1px outset var(--face); max-height: 140px; overflow-y: auto;
        }
        .mention-option {
          display: flex; align-items: center; gap: 6px; width: 100%; background: transparent; border: none;
          padding: 5px 8px; cursor: pointer; font-family: inherit; font-size: 11px; text-align: left;
          border-bottom: 1px solid #E5E2D6;
        }
        .mention-option:hover { background: var(--win-blue-light); }

        .cheatsheet-section { margin: 14px 0 18px; }
        .cheatsheet-list { display: flex; flex-direction: column; gap: 6px; margin-bottom: 8px; }
        .cheatsheet-row { display: flex; align-items: flex-start; gap: 10px; background: var(--face); border: 1px solid #C3C8CC; padding: 8px 10px; }
        .cheatsheet-icon { color: var(--win-blue); flex-shrink: 0; margin-top: 1px; }
        .cheatsheet-main { flex: 1; min-width: 0; }
        .cheatsheet-title { font-size: 11.5px; font-weight: bold; margin-bottom: 2px; }
        .cheatsheet-link { font-size: 11px; color: var(--win-blue); text-decoration: underline; word-break: break-all; }
        .cheatsheet-note { font-size: 11px; line-height: 1.4; }
        .cheatsheet-meta { display: flex; align-items: center; gap: 5px; flex-shrink: 0; }
        .inline-textarea { min-height: 50px; resize: vertical; font-family: inherit; }
        .file-upload-btn {
          display: flex; align-items: center; gap: 6px; background: var(--content-bg); border: 1px inset var(--face);
          color: var(--text-primary); font-size: 11px; padding: 6px 8px; cursor: pointer;
        }
      `}</style>

      <div className="titlebar">
        <div className="titlebar-left">
          <div className="titlebar-icon">R</div>
          <span className="titlebar-title">R&amp;D Ops — PT USSI Pinbuk Prima Software</span>
        </div>
        <div className="titlebar-btns">
          <button className="titlebar-btn">–</button>
          <button className="titlebar-btn">▢</button>
          <button className="titlebar-btn">✕</button>
        </div>
      </div>

      <div className="topbar">
        <div className="tabs">
          <button
            className={`tab-btn ${tab === "dashboard" ? "tab-btn-active" : ""}`}
            onClick={() => setTab("dashboard")}
          >
            <LayoutDashboard size={13} aria-hidden="true" /> Dashboard
          </button>
          <button
            className={`tab-btn ${tab === "boards" ? "tab-btn-active" : ""}`}
            onClick={() => setTab("boards")}
          >
            <LayoutGrid size={13} aria-hidden="true" /> Boards
          </button>
          <button
            className={`tab-btn ${tab === "weekly" ? "tab-btn-active" : ""}`}
            onClick={() => setTab("weekly")}
          >
            <CalendarClock size={13} aria-hidden="true" /> My Weekly Plan
          </button>
          <button
            className={`tab-btn ${tab === "review" ? "tab-btn-active" : ""}`}
            onClick={() => setTab("review")}
          >
            <ClipboardCheck size={13} aria-hidden="true" /> Review queue
            {pendingReview > 0 && <span className="review-badge">{pendingReview}</span>}
          </button>
        </div>
        <div className="topbar-right">
          <div className="notif-wrap">
            <button
              className="icon-only-btn"
              onClick={() => {
                setShowNotif((s) => !s);
                setShowUserMenu(false);
              }}
              aria-label="Notifikasi"
            >
              <Bell size={15} aria-hidden="true" />
              {pendingReview > 0 && <span className="notif-badge">{pendingReview}</span>}
            </button>
            {showNotif && (
              <div className="dropdown-panel notif-panel">
                <div className="dropdown-title">Butuh atensi lo</div>
                {cards.filter((c) => !reviewedState[c.id]).slice(0, 5).map((c) => (
                  <div className="notif-item" key={c.id}>
                    <span className="review-dot" />
                    <span className="small">{c.title}</span>
                  </div>
                ))}
                {pendingReview === 0 && <div className="empty-note">Ga ada yang perlu direview. Rapi.</div>}
                <button
                  className="quick-btn"
                  style={{ width: "100%", marginTop: 6 }}
                  onClick={() => {
                    setTab("review");
                    setShowNotif(false);
                  }}
                >
                  Buka Review Queue
                </button>
              </div>
            )}
          </div>
          <div className="user-menu-wrap">
            <button
              className="user-menu-trigger"
              onClick={() => {
                setShowUserMenu((s) => !s);
                setShowNotif(false);
              }}
            >
              <Avatar id="lugina" size={26} />
            </button>
            {showUserMenu && (
              <div className="dropdown-panel user-menu-panel">
                <div className="user-menu-head">
                  <Avatar id="lugina" size={30} />
                  <div>
                    <div className="panel-row-name">{profileName}</div>
                    <div className="muted small">SPV &amp; Developer</div>
                  </div>
                </div>
                <div className="dropdown-divider" />
                <button className="dropdown-item" onClick={() => { setShowProfile(true); setShowUserMenu(false); }}>
                  <UserRound size={13} aria-hidden="true" /> My Profile
                </button>
                <button className="dropdown-item" onClick={() => { setShowSettings(true); setSettingsTab("users"); setShowUserMenu(false); }}>
                  <Settings size={13} aria-hidden="true" /> Settings
                </button>
                <button className="dropdown-item" onClick={() => window.alert("Panduan penggunaan R&D Ops — dokumentasi internal (demo).")}>
                  <HelpCircle size={13} aria-hidden="true" /> Help
                </button>
                <div className="dropdown-divider" />
                <button className="dropdown-item dropdown-item-danger" onClick={() => { setSignedOut(true); setShowUserMenu(false); }}>
                  <LogOut size={13} aria-hidden="true" /> Sign out
                </button>
              </div>
            )}
          </div>
        </div>
      </div>

      <div className="content">
        {tab === "dashboard" && <Dashboard cards={cards} onOpenTask={setOpenTask} />}
        {tab === "boards" && (
          <BoardsView
            activeBoard={activeBoard}
            setActiveBoard={setActiveBoard}
            dailyTasks={dailyTasks}
            onAddDailyTask={addDailyTask}
            onUpdateDay={updateDayField}
            onToggleDone={toggleDayDone}
            onAddBigTask={addBigTask}
            extraBigTasks={extraBigTasks}
            comments={comments}
            onAddComment={addComment}
            cheatSheet={cheatSheet}
            onAddCheatSheet={addCheatSheetItem}
            signedBigTasks={signedBigTasks}
            onSignBigTask={signBigTask}
            onUnsignBigTask={unsignBigTask}
          />
        )}
        {tab === "weekly" && (
          <WeeklyPlanView
            allBigTasks={[...BIG_TASKS, ...extraBigTasks]}
            dailyTasks={dailyTasks}
            weekStart={weekStart}
            onPrevWeek={() => setWeekStart((w) => shiftWeek(w, -1))}
            onNextWeek={() => setWeekStart((w) => shiftWeek(w, 1))}
            weeklyPushState={weeklyPushState}
            onPushWeekly={pushWeekly}
          />
        )}
        {tab === "review" && (
          <ReviewQueue
            cards={cards}
            reviewedState={reviewedState}
            setReviewedState={setReviewedState}
            onOpenTask={setOpenTask}
          />
        )}
      </div>

      {openTask && (
        <TaskDetail
          card={openTask}
          onClose={() => setOpenTask(null)}
          reviewedState={reviewedState}
          setReviewedState={setReviewedState}
        />
      )}
      {showProfile && (
        <ProfileModal
          onClose={() => setShowProfile(false)}
          name={profileName}
          setName={setProfileName}
          initials={profileInitials}
          setInitials={setProfileInitials}
        />
      )}
      {showSettings && (
        <SettingsModal
          onClose={() => setShowSettings(false)}
          settingsTab={settingsTab}
          setSettingsTab={setSettingsTab}
          theme={theme}
          setTheme={setTheme}
        />
      )}
    </div>
  );
}
