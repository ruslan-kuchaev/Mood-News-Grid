import { type FormEvent } from 'react'

interface SearchInputProps {
  value: string
  onChange: (value: string) => void
  onSubmit: () => void
  loading: boolean
  variant: 'center' | 'top'
}

function SearchInput({ value, onChange, onSubmit, loading, variant }: SearchInputProps) {
  function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    if (!value.trim() || loading) return
    onSubmit()
  }

  return (
    <form
      onSubmit={handleSubmit}
      className={
        variant === 'center'
          ? 'w-full max-w-xl px-4'
          : 'sticky top-6 z-10 w-full max-w-xl mx-auto px-4'
      }
    >
      <div className="flex items-center gap-2 rounded-full bg-neutral-900 border border-neutral-700 px-5 py-3 shadow-lg shadow-black/40 focus-within:border-neutral-400 transition-colors">
        <input
          type="text"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder="Найти новости..."
          disabled={loading}
          autoFocus={variant === 'center'}
          className="flex-1 bg-transparent text-neutral-100 placeholder-neutral-500 outline-none disabled:opacity-50"
        />
        <button
          type="submit"
          disabled={loading || !value.trim()}
          className="text-sm font-medium text-neutral-300 hover:text-white disabled:text-neutral-600 disabled:cursor-not-allowed transition-colors"
        >
          {loading ? '...' : 'Найти'}
        </button>
      </div>
    </form>
  )
}

export default SearchInput
