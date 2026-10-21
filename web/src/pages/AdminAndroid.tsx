import { useCallback, useEffect, useState } from 'react'
import { adminApi, type AndroidReleaseView } from '../api'
import { Field, FormShell, PageHeader, Status } from '../components/ui'
import { clockTime, longDate } from '../format'

const SOURCE: Record<AndroidReleaseView['source'], string> = {
  database: 'Diatur di halaman ini',
  env: 'Dari pengaturan server (ANDROID_LATEST_VERSION / ANDROID_DOWNLOAD_URL)',
  none: 'Belum diatur — aplikasi tidak menampilkan pemberitahuan versi baru',
}

/**
 * Admins set the newest Android app version and its download link. Installed
 * apps older than it show "Versi baru tersedia" with a download button.
 */
export default function AdminAndroid() {
  const [view, setView] = useState<AndroidReleaseView | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [version, setVersion] = useState('')
  const [url, setUrl] = useState('')

  const show = useCallback((v: AndroidReleaseView) => {
    setView(v)
    setVersion(v.latestVersion)
    setUrl(v.downloadUrl)
  }, [])
  useEffect(() => {
    adminApi.android().then(show, (e: Error) => setError(e.message))
  }, [show])

  return (
    <>
      <PageHeader
        title="Aplikasi Android"
        subtitle="Versi terbaru yang ditawarkan kepada pengguna aplikasi Jemaat Siloam"
      />
      <Status loading={!view && !error} error={error} />
      {view && (
        <>
          <section className="card panel">
            <header className="panel-head">
              <h2>Versi saat ini</h2>
            </header>
            <dl className="details release-now">
              <dt>Versi terbaru</dt>
              <dd className="strong">{view.latestVersion || <span className="muted">–</span>}</dd>
              <dt>Tautan unduhan</dt>
              <dd>
                {view.downloadUrl ? (
                  <a href={view.downloadUrl} target="_blank" rel="noreferrer">
                    {view.downloadUrl}
                  </a>
                ) : (
                  <span className="muted">–</span>
                )}
              </dd>
              <dt>Sumber</dt>
              <dd>
                {SOURCE[view.source]}
                {view.source === 'database' && view.updatedAt && (
                  <span className="muted">
                    {' '}
                    · {view.updatedBy || 'admin'}, {longDate(view.updatedAt)} {clockTime(view.updatedAt)}
                  </span>
                )}
              </dd>
            </dl>
          </section>

          <section className="card panel">
            <header className="panel-head">
              <h2>Ubah versi</h2>
              <p className="muted">
                Aplikasi yang lebih lama dari versi ini menampilkan notifikasi (sekali per versi) dan tombol “Unduh”.
                Naikkan dulu <code>versionName</code> di <code>android/app/build.gradle.kts</code>, tanda tangani APK
                dengan kunci yang sama, lalu unggah APK-nya, misalnya ke GitHub Releases.
              </p>
            </header>
            <FormShell
              submitLabel="Simpan"
              successMessage="Tersimpan. Aplikasi akan melihatnya pada pemeriksaan berikutnya (paling lama sekitar 1 jam)."
              onSubmit={async () =>
                show(await adminApi.setAndroid({ latestVersion: version.trim(), downloadUrl: url.trim() }))
              }
            >
              <Field label="Versi terbaru (mis. 1.0.1)">
                <input
                  value={version}
                  onChange={(e) => setVersion(e.target.value)}
                  placeholder="1.0.1"
                  inputMode="decimal"
                />
              </Field>
              <Field label="Tautan unduhan APK (https://)" wide>
                <input
                  type="url"
                  value={url}
                  onChange={(e) => setUrl(e.target.value)}
                  placeholder="https://github.com/michaelwp/siloam/releases/latest/download/jemaat-siloam.apk"
                />
              </Field>
              <p className="wide muted small">Kosongkan keduanya untuk mematikan pemberitahuan versi baru.</p>
            </FormShell>
            {view.source === 'database' && (
              <p className="panel-foot">
                <button
                  type="button"
                  className="link"
                  onClick={() => adminApi.clearAndroid().then(show, (e: Error) => setError(e.message))}
                >
                  Hapus pengaturan ini dan gunakan pengaturan server
                  {view.server.latestVersion ? ` (versi ${view.server.latestVersion})` : ' (tidak ada)'}
                </button>
              </p>
            )}
          </section>
        </>
      )}
    </>
  )
}
