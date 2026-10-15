import { ref, watch } from 'vue';
import { useReaderPersistence } from './useReaderPersistence';

export function useTheme() {
  const { readDarkMode, writeDarkMode } = useReaderPersistence();
  const isDarkMode = ref(readDarkMode());

  watch(isDarkMode, (value) => {
    writeDarkMode(value);
  });

  function toggleTheme(): void {
    isDarkMode.value = !isDarkMode.value;
  }

  return { isDarkMode, toggleTheme };
}
