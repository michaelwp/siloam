import { useState, type CSSProperties } from 'react'
import type { Member } from '../api'
import { MonthGrid, MonthNav } from '../components/calendar'
import { MemberDialogs, type MemberDialog } from '../components/forms'
import { ActionsHeader, EditOnly, Modal, PageHeader, RowActions, Search, Status } from '../components/ui'
import { useCanEdit, useFullView } from '../auth'
import { useData } from '../data'
import { anniversaryIn, birthLine, byText, dayMonth, matches, MONTHS } from '../format'
import { dayKey, longDay, monthDays, type Cursor } from '../schedule'

type View = 'list' | 'calendar'
type Shown = { kind: 'member'; member: Member } | { kind: 'day'; key: string } | null

// the birthday color in the calendar (as the jenis colors in Jadwal Ibadah)
const BIRTHDAY_COLOR = '#d9468f'

/** Birthdays per day ("2026-10-22") of the days a month's grid shows; 29 Feb falls on the 28th in other years. */
function birthdaysByDay(members: Member[], cursor: Cursor) {
  const map = new Map(monthDays(cursor.y, cursor.m).map((day) => [day.key, [] as Member[]]))
  for (const m of members) {
    if (!m.birthMonth || !m.birthDay) continue
    // the grid spans the month before and after too
    for (const y of [cursor.y - 1, cursor.y, cursor.y + 1]) {
      const d = anniversaryIn(y, m.birthMonth, m.birthDay)
      map.get(dayKey(d.getFullYear(), d.getMonth(), d.getDate()))?.push(m)
    }
  }
  for (const [k, list] of map) {
    if (list.length) map.set(k, list.toSorted((a, b) => byText(a.name, b.name)))
    else map.delete(k)
  }
  return map
}

export default function HutJemaat() {
  const full = useFullView()
  const canEdit = useCanEdit()
  const { members, loading, error } = useData()
  const [q, setQ] = useState('')
  const [month, setMonth] = useState<number | 'all'>('all')
  const [dialog, setDialog] = useState<MemberDialog>(null)
  const [today] = useState(() => new Date())
  const thisMonth = today.getMonth()
  const [view, setView] = useState<View>('list')
  const [cursor, setCursor] = useState<Cursor>(() => ({ y: today.getFullYear(), m: today.getMonth() }))
  const [shown, setShown] = useState<Shown>(null)

  // birthMonth/birthDay are present for visitors too (the full date is not)
  const withDate = members.filter((m) => m.birthMonth && matches(q, m.name, m.familyName))
  const noDate = members.filter((m) => !m.birthMonth)
  const months = MONTHS.map((name, i) => ({
    name,
    i,
    rows: withDate
      .filter((m) => m.birthMonth === i + 1)
      .toSorted((a, b) => a.birthDay! - b.birthDay!),
  })).filter((g) => month === 'all' || g.i === month)
  const byDay = birthdaysByDay(withDate, cursor)
  const todayKey = dayKey(today.getFullYear(), today.getMonth(), today.getDate())
  // switching views keeps the month: the list's chosen month opens in the calendar, and back
  const showView = (v: View) => {
    if (v === 'calendar' && month !== 'all') setCursor({ y: today.getFullYear(), m: month })
    if (v === 'list' && month !== 'all') setMonth(cursor.m)
    setView(v)
  }

  return (
    <>
      <PageHeader
        title="Daftar HUT Jemaat"
        subtitle={`${members.length} jiwa, ${view === 'list' ? 'dikelompokkan per bulan kelahiran' : 'per tanggal ulang tahun'}`}
      >
        <Search value={q} onChange={setQ} />
        <fieldset className="segmented">
          <legend className="sr-only">Tampilan</legend>
          <button aria-pressed={view === 'list'} onClick={() => showView('list')}>
            Daftar
          </button>
          <button aria-pressed={view === 'calendar'} onClick={() => showView('calendar')}>
            Kalender
          </button>
        </fieldset>
        {view === 'list' && (
          <select value={month} onChange={(e) => setMonth(e.target.value === 'all' ? 'all' : Number(e.target.value))}>
            <option value="all">Semua bulan</option>
            {MONTHS.map((m, i) => (
              <option key={m} value={i}>
                {m}
              </option>
            ))}
          </select>
        )}
        <EditOnly>
          <button className="btn primary" onClick={() => setDialog({ kind: 'add' })}>
            + Tambah Anggota
          </button>
        </EditOnly>
      </PageHeader>

      <Status loading={loading} error={error} empty={!members.length} />
      {!loading && !error && view === 'calendar' && (
        <section className="cal-main">
          <MonthNav cursor={cursor} today={today} subtitle={`${byMonth(byDay, cursor)} ulang tahun`} onChange={setCursor} />
          <MonthGrid
            cursor={cursor}
            todayKey={todayKey}
            byDay={byDay}
            itemKey={(m) => m.id}
            renderItem={(m) => (
              <button
                className="cal-event"
                style={{ '--type-color': BIRTHDAY_COLOR } as CSSProperties}
                title={`${m.name} · Kel. ${m.familyName}`}
                onClick={() => setShown({ kind: 'member', member: m })}
              >
                <span className="cal-dot" aria-hidden="true" />
                <span className="cal-event-title">{m.name}</span>
              </button>
            )}
            onMore={(key) => setShown({ kind: 'day', key })}
            // signed in: the empty part of a day adds a member born on that day (and month)
            onAdd={canEdit ? (key) => setDialog({ kind: 'add', birthDate: key }) : undefined}
            addLabel="Tambah anggota yang lahir"
          />
        </section>
      )}
      {!loading &&
        view === 'list' &&
        months.map((g) => (
          <section key={g.name} className={`card month ${g.i === thisMonth ? 'current' : ''}`}>
            <header className="family-head">
              <h2>
                {g.name.toUpperCase()} {g.i === thisMonth && <span className="badge green">Bulan ini</span>}
              </h2>
              <span className="muted">{g.rows.length} orang</span>
            </header>
            {g.rows.length > 0 && (
              <div className="table-wrap flat">
                <table>
                  <thead>
                    <tr>
                      <th className="num">No</th>
                      <th>Nama Lengkap</th>
                      <th>{full ? 'Tempat, Tanggal Lahir' : 'Ulang Tahun'}</th>
                      {full && <th>Umur</th>}
                      <th>Nama Keluarga</th>
                      <ActionsHeader />
                    </tr>
                  </thead>
                  <tbody>
                    {g.rows.map((m, i) => (
                      <tr key={m.id}>
                        <td className="num">{i + 1}</td>
                        <td className="strong">{m.name}</td>
                        <td>{full ? birthLine(m) : dayMonth(m.birthDay, m.birthMonth)}</td>
                        {full && <td className="nowrap">{m.ageText}</td>}
                        <td>Kel. {m.familyName}</td>
                        <RowActions
                          onEdit={() => setDialog({ kind: 'edit', member: m })}
                          onDelete={() => setDialog({ kind: 'delete', member: m })}
                        />
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>
        ))}

      {shown?.kind === 'member' && (
        <BirthdayDetails
          member={shown.member}
          full={full}
          onClose={() => setShown(null)}
          onEdit={() => setDialog({ kind: 'edit', member: shown.member })}
        />
      )}
      {shown?.kind === 'day' && (
        <Modal title={`Ulang tahun · ${longDay(shown.key)}`} onClose={() => setShown(null)}>
          <ul className="day-list">
            {(byDay.get(shown.key) ?? []).map((m) => (
              <li key={m.id}>
                <button
                  className="cal-event"
                  style={{ '--type-color': BIRTHDAY_COLOR } as CSSProperties}
                  onClick={() => setShown({ kind: 'member', member: m })}
                >
                  <span className="cal-dot" aria-hidden="true" />
                  <span className="cal-event-title">{m.name}</span>{' '}
                  <span className="muted">· Kel. {m.familyName}</span>
                </button>
              </li>
            ))}
          </ul>
        </Modal>
      )}

      {!loading && view === 'list' && noDate.length > 0 && (
        <section className="card">
          <header className="family-head">
            <h2>Tanggal lahir belum diisi</h2>
          </header>
          <ul className="plain-list">
            {noDate.map((m) => (
              <li key={m.id}>
                {m.name} <span className="muted">(Kel. {m.familyName})</span>{' '}
                <EditOnly>
                  <button className="link" onClick={() => setDialog({ kind: 'edit', member: m })}>
                    Lengkapi
                  </button>
                </EditOnly>
              </li>
            ))}
          </ul>
        </section>
      )}
      <MemberDialogs dialog={dialog} onClose={() => setDialog(null)} />
    </>
  )
}

/** How many birthdays fall in the cursor's month (the grid also shows days of the months around it). */
function byMonth(byDay: Map<string, Member[]>, { y, m }: Cursor) {
  const prefix = dayKey(y, m, 1).slice(0, 8)
  let n = 0
  for (const [k, list] of byDay) if (k.startsWith(prefix)) n += list.length
  return n
}

/** One birthday from the calendar: the day and month (birth place, date and age when signed in). */
function BirthdayDetails({
  member: m,
  full,
  onClose,
  onEdit,
}: {
  member: Member
  full: boolean
  onClose: () => void
  onEdit: () => void
}) {
  const rows: [string, string][] = [
    ['Ulang tahun', dayMonth(m.birthDay, m.birthMonth)],
    ['Tempat, tanggal lahir', full ? birthLine(m) : ''],
    ['Umur', full ? m.ageText : ''],
    ['Keluarga', `Kel. ${m.familyName}`],
    ['Kolom', String(m.kolom)],
  ]
  return (
    <Modal title="Ulang Tahun" onClose={onClose}>
      <div className="event-details" style={{ '--type-color': BIRTHDAY_COLOR } as CSSProperties}>
        <div className="event-head">
          <span className="event-swatch" aria-hidden="true" />
          <div>
            <h3>🎂 {m.name}</h3>
          </div>
        </div>
        <dl className="event-rows">
          {rows
            .filter(([, v]) => v)
            .map(([k, v]) => (
              <div key={k}>
                <dt>{k}</dt>
                <dd>{v}</dd>
              </div>
            ))}
        </dl>
      </div>
      <EditOnly>
        <footer className="modal-foot">
          <button
            className="btn primary"
            onClick={() => {
              onClose()
              onEdit()
            }}
          >
            Ubah
          </button>
        </footer>
      </EditOnly>
    </Modal>
  )
}
