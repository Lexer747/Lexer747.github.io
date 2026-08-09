// Target all carousels on the page
const carousels = document.querySelectorAll('.image-carousel');

carousels.forEach((carousel) => {
    let scrollInterval;
    const seed = Math.random() * 1.8;
    function startScroll() {
        scrollInterval = setInterval(() => {
            // Reset to 0 when reaching halfway point (assuming duplicated items)
            if (carousel.scrollLeft >= (carousel.scrollWidth / 2)) {
                carousel.scrollLeft = 0;
            } else {
                carousel.scrollLeft += 0.8 + seed;
            }
        }, 28); // Speed control (lower = faster)
    }

    function stopScroll() {
        clearInterval(scrollInterval);
    }

    // Initialize the scrolling for this specific carousel
    startScroll();

    // Bind individual hover listeners to pause/play independently
    carousel.addEventListener('mouseenter', stopScroll);
    carousel.addEventListener('mouseleave', startScroll);
});