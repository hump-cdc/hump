/** Counter making every `importFresh` specifier unique. */
let freshImports = 0

/**
 * Loads a new instance of the module at `url` (pass `import.meta.resolve('./module')`), bypassing
 * the module cache bun shares between test files. Its load-time code reruns, e.g. a persisted store
 * binding to the `localStorage` of the DOM registered now.
 *
 * The coverage report keeps only the instance of a file loaded last, which is this one. So a test
 * file that uses it imports the module statically as well (that instance loads first, whatever order
 * the files run in), loads one fresh instance, and runs every function of the module on it.
 */
export const importFresh = async <T>(url: string): Promise<T> => {
    freshImports += 1
    return import(`${Bun.fileURLToPath(url)}?fresh=${freshImports}`)
}
