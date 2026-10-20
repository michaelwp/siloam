import { useEffect, useRef, useState, type ReactNode } from 'react'

// Small SVG charts following the dataviz method: thin marks (<= 24px, 4px
// rounded data-end, square at the baseline), 2px surface gaps, hairline solid
// grid, text in text tokens (never the series color), per-mark tooltips on
// hover and keyboard focus, and a table view so no value is tooltip-only.
// Colors come from CSS tokens (--series-1/2, --viz-deemph), validated for the
// light and dark card surfaces.

const BAR_MAX = 24
const RADIUS = 4

/** Width of the element, kept in sync with resizes. */
function useWidth<T extends HTMLElement>() {
  const ref = useRef<T>(null)
  const [width, setWidth] = useState(0)
  useEffect(() => {
    const el = ref.current
    if (!el) return
    const ro = new ResizeObserver(([entry]) => setWidth(Math.floor(entry.contentRect.width)))
    ro.observe(el)
    return () => ro.disconnect()
  }, [])
  return [ref, width] as const
}

/** Clean axis ticks (steps of 1, 2 or 5 × 10^n) covering 0..max. */
function niceTicks(max: number, target = 4) {
  const raw = Math.max(max, 1) / target
  const pow = 10 ** Math.floor(Math.log10(raw))
  const step = [1, 2, 5, 10].map((m) => m * pow).find((s) => s >= raw) ?? raw
  const top = Math.max(step, Math.ceil(max / step) * step)
  const ticks: number[] = []
  for (let v = 0; v <= top + 1e-9; v += step) ticks.push(Math.round(v * 100) / 100)
  return { ticks, top }
}

/** Column: rounded top (data-end), square at the baseline. */
function columnPath(x: number, y: number, w: number, h: number) {
  const r = Math.min(RADIUS, w / 2, h)
  return `M${x},${y + h}V${y + r}Q${x},${y} ${x + r},${y}H${x + w - r}Q${x + w},${y} ${x + w},${y + r}V${y + h}Z`
}

/** Horizontal bar: rounded right end (data-end), square at the left. */
function barPath(x: number, y: number, w: number, h: number) {
  const r = Math.min(RADIUS, h / 2, w)
  return `M${x},${y}H${x + w - r}Q${x + w},${y} ${x + w},${y + r}V${y + h - r}Q${x + w},${y + h} ${x + w - r},${y + h}H${x}Z`
}

type Tip = { x: number; y: number; title: string; rows: { key?: string; label: string; value: number }[] } | null

/** One tooltip per chart: value leads, label follows, line-key for series. */
function Tooltip({ tip }: { tip: Tip }) {
  if (!tip) return null
  return (
    <output className="viz-tooltip" style={{ left: tip.x, top: tip.y }}>
      <div className="viz-tooltip-title">{tip.title}</div>
      {tip.rows.map((r) => (
        <div key={r.label} className="viz-tooltip-row">
          {r.key && <span className={`viz-key ${r.key}`} aria-hidden="true" />}
          <strong>{r.value}</strong>
          <span>{r.label}</span>
        </div>
      ))}
    </output>
  )
}

/** Collapsible table twin of a chart. */
function TableView({ head, rows }: { head: string[]; rows: (string | number)[][] }) {
  return (
    <details className="viz-table">
      <summary>Lihat tabel</summary>
      <table>
        <thead>
          <tr>
            {head.map((h, i) => (
              <th key={h} className={i > 0 ? 'num' : undefined}>
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={String(r[0])}>
              {r.map((c, i) => (
                <td key={i} className={i > 0 ? 'num' : undefined}>
                  {c}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </details>
  )
}

export function ChartCard({ title, subtitle, children }: { title: string; subtitle?: ReactNode; children: ReactNode }) {
  return (
    <section className="card chart-card">
      <header className="chart-head">
        <h2>{title}</h2>
        {subtitle && <p className="muted small">{subtitle}</p>}
      </header>
      {children}
    </section>
  )
}

export type Column = { label: string; short: string; value: number; emphasis?: boolean }

/**
 * Vertical columns for one series. With `emphasize`, emphasized columns are in
 * the accent and the rest in the de-emphasis gray (the "this one" story);
 * otherwise every column is series 1. Only emphasized columns (or, without
 * emphasis, the largest) carry a value label; the tooltip and table carry the rest.
 */
export function ColumnChart({
  data,
  unit,
  emphasize = false,
  labelHead,
}: {
  data: Column[]
  unit: string
  emphasize?: boolean
  labelHead: string
}) {
  const [ref, width] = useWidth<HTMLDivElement>()
  const [tip, setTip] = useState<Tip>(null)
  const height = 180
  const m = { top: 18, right: 4, bottom: 26, left: 28 }
  const plotW = Math.max(width - m.left - m.right, 0)
  const plotH = height - m.top - m.bottom
  const { ticks, top } = niceTicks(Math.max(...data.map((d) => d.value), 0))
  const band = data.length ? plotW / data.length : 0
  const barW = Math.min(BAR_MAX, band * 0.62)
  const y = (v: number) => m.top + plotH - (v / top) * plotH
  const maxValue = Math.max(...data.map((d) => d.value))
  const labelled = (d: Column) => (emphasize ? !!d.emphasis : d.value === maxValue) && d.value > 0

  return (
    <>
      <div className="viz" ref={ref} onPointerLeave={() => setTip(null)}>
        {width > 0 && (
          <svg width={width} height={height}>
            <title>{`${labelHead}: ${data.map((d) => `${d.label} ${d.value}`).join(', ')}`}</title>
            {ticks.map((t) => (
              <g key={t}>
                <line className={t === 0 ? 'viz-baseline' : 'viz-grid'} x1={m.left} x2={width - m.right} y1={y(t)} y2={y(t)} />
                <text className="viz-tick" x={m.left - 6} y={y(t)} dy="0.32em" textAnchor="end">
                  {t}
                </text>
              </g>
            ))}
            {data.map((d, i) => {
              const cx = m.left + band * i + band / 2
              const h = y(0) - y(d.value)
              const show = () =>
                // above the value label (which sits 5px over the cap), never on top of it
                setTip({ x: cx, y: y(d.value) - (labelled(d) ? 22 : 8), title: d.label, rows: [{ label: unit, value: d.value }] })
              return (
                <g key={d.label}>
                  {d.value > 0 && (
                    <path
                      className={emphasize && !d.emphasis ? 'viz-deemph' : 'viz-s1'}
                      d={columnPath(cx - barW / 2, y(d.value), barW, h)}
                    />
                  )}
                  {labelled(d) && (
                    <text className="viz-value" x={cx} y={y(d.value) - 5} textAnchor="middle">
                      {d.value}
                    </text>
                  )}
                  <text className="viz-tick" x={cx} y={height - 8} textAnchor="middle">
                    {d.short}
                  </text>
                  {/* hit target: the whole band, bigger than the mark */}
                  <rect
                    className="viz-hit"
                    x={m.left + band * i}
                    y={m.top}
                    width={band}
                    height={plotH}
                    tabIndex={0}
                    aria-label={`${d.label}: ${d.value} ${unit}`}
                    onPointerEnter={show}
                    onFocus={show}
                    onBlur={() => setTip(null)}
                  />
                </g>
              )
            })}
          </svg>
        )}
        <Tooltip tip={tip} />
      </div>
      <TableView head={[labelHead, unit]} rows={data.map((d) => [d.label, d.value])} />
    </>
  )
}

export type StackRow = { label: string; values: number[] }
export type Series = { name: string; cls: string }

/**
 * Horizontal stacked bars (part-to-whole per row) with a legend, a total at
 * each bar end, 2px surface gaps between segments and per-segment tooltips.
 */
export function StackedBarChart({ rows, series, rowHead }: { rows: StackRow[]; series: Series[]; rowHead: string }) {
  const [ref, width] = useWidth<HTMLDivElement>()
  const [tip, setTip] = useState<Tip>(null)
  const rowH = 36
  const m = { top: 4, right: 34, bottom: 4, left: 92 }
  const height = m.top + rows.length * rowH + m.bottom
  const plotW = Math.max(width - m.left - m.right, 0)
  const maxTotal = Math.max(...rows.map((r) => r.values.reduce((a, b) => a + b, 0)), 1)
  const scale = (v: number) => (v / maxTotal) * plotW
  const GAP = 2

  return (
    <>
      <ul className="viz-legend">
        {series.map((s) => (
          <li key={s.name}>
            <span className={`viz-swatch ${s.cls}`} aria-hidden="true" />
            {s.name}
          </li>
        ))}
      </ul>
      <div className="viz" ref={ref} onPointerLeave={() => setTip(null)}>
        {width > 0 && (
          <svg width={width} height={height}>
            <title>{`${rowHead} per ${series.map((s) => s.name).join(' dan ')}`}</title>
            <line className="viz-baseline" x1={m.left} x2={m.left} y1={m.top} y2={height - m.bottom} />
            {rows.map((r, i) => {
              const total = r.values.reduce((a, b) => a + b, 0)
              const yTop = m.top + i * rowH + (rowH - BAR_MAX) / 2
              const last = r.values.reduce((acc, v, k) => (v > 0 ? k : acc), -1)
              let x = m.left
              const segments = r.values.map((v, k) => {
                const w = scale(v)
                const seg = { k, v, x, w: k === last ? w : Math.max(w - GAP, 0) }
                x += w
                return seg
              })
              const show = (k: number) => () =>
                setTip({
                  x: m.left + Math.min(scale(total), plotW) / 2,
                  y: yTop - 6,
                  title: `${r.label} · ${total} jiwa`,
                  rows: series.map((s, j) => ({ key: s.cls, label: s.name, value: r.values[j] })).filter((_, j) => j === k || k < 0),
                })
              return (
                <g key={r.label}>
                  <text className="viz-label" x={m.left - 8} y={yTop + BAR_MAX / 2} dy="0.32em" textAnchor="end">
                    {r.label}
                  </text>
                  {segments.map(
                    (s) =>
                      s.v > 0 &&
                      (s.k === last ? (
                        <path key={s.k} className={series[s.k].cls} d={barPath(s.x, yTop, s.w, BAR_MAX)} />
                      ) : (
                        <rect key={s.k} className={series[s.k].cls} x={s.x} y={yTop} width={s.w} height={BAR_MAX} />
                      )),
                  )}
                  <text className="viz-value" x={m.left + scale(total) + 6} y={yTop + BAR_MAX / 2} dy="0.32em">
                    {total}
                  </text>
                  {/* hit targets: each segment's full row height (bigger than the mark) */}
                  {segments.map(
                    (s) =>
                      s.v > 0 && (
                        <rect
                          key={`hit-${s.k}`}
                          className="viz-hit"
                          x={s.x}
                          y={m.top + i * rowH}
                          width={Math.max(scale(s.v), 1)}
                          height={rowH}
                          tabIndex={0}
                          aria-label={`${r.label}, ${series[s.k].name}: ${s.v}`}
                          onPointerEnter={show(s.k)}
                          onFocus={show(s.k)}
                          onBlur={() => setTip(null)}
                        />
                      ),
                  )}
                </g>
              )
            })}
          </svg>
        )}
        <Tooltip tip={tip} />
      </div>
      <TableView
        head={[rowHead, ...series.map((s) => s.name), 'Jumlah']}
        rows={rows.map((r) => [r.label, ...r.values, r.values.reduce((a, b) => a + b, 0)])}
      />
    </>
  )
}
