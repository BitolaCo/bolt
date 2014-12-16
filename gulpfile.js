var gulp = require("gulp"),
    rename = require("gulp-rename"),
    uglify = require("gulp-uglify"),
    base = __dirname + "/assets";

gulp.task("watch", function() {
    gulp.watch(base + "/client.js", ["uglify"]);
});

gulp.task("uglify", function() {
    return gulp.src(base + "/client.js")
        .pipe(uglify())
        .pipe(rename("client.min.js"))
        .pipe(gulp.dest(base));
});

gulp.task("default", ["uglify"]);