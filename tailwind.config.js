/** @type {import('tailwindcss').Config} */
module.exports = {
    future: {},
    purge: [],
    theme: {
        extend: {},
    },
    variants: {},
    plugins: [
        require('tailwindcss')
    ],
    content: [
        "./build/**/*.html",
    ],
}

