Published.content and folder name form the ending url for a blog post.

If either changes you can add a redirect to the new URL by creating a template file at the original source
location. Which has the `{{blog-redirect:acci-ping.html}}` ending output blog html file as an attribute. This
will cause the current template file to emit the URL for the new location of that file.