export function debounce(fn, delay) {
    let timeoutId;
    function debounced(...args) {
        if (timeoutId) clearTimeout(timeoutId);
        timeoutId = setTimeout(() => {
           fn(...args);
        }, delay);
    }
    debounced.cancel = () => { if (timeoutId) { clearTimeout(timeoutId); timeoutId = null } }
    return debounced
}
