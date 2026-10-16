import { onBeforeUnmount, ref, watch, type Ref } from 'vue'
import { http, qs } from '@/lib/http'

export interface AutocompleteOption { value: number; label: string; kode?: string }

export function useAutocomplete(endpoint: string, query: Ref<string>, delay = 300) {
  const options = ref<AutocompleteOption[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  let timer: ReturnType<typeof setTimeout> | undefined
  let controller: AbortController | undefined

  const stop = watch(query, (value) => {
    clearTimeout(timer); controller?.abort()
    const term = value.trim()
    if (!term) { options.value = []; loading.value = false; error.value = null; return }
    timer = setTimeout(async () => {
      controller = new AbortController(); loading.value = true; error.value = null
      try { options.value = await http.get<AutocompleteOption[]>(`${endpoint}${qs({ q: term })}`, { signal: controller.signal }) }
      catch (reason) { if (!(reason instanceof DOMException && reason.name === 'AbortError')) error.value = reason instanceof Error ? reason.message : 'Pencarian gagal.' }
      finally { if (!controller?.signal.aborted) loading.value = false }
    }, delay)
  })
  onBeforeUnmount(() => { stop(); clearTimeout(timer); controller?.abort() })
  return { options, loading, error }
}
