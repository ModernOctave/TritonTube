// Lab 7, 8, 9: Use these templates to render the web pages

package web

const indexHTML = `
<!DOCTYPE html>
<html>
  <head>
    <meta charset="UTF-8" />
    <title>TritonTube</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.8/dist/css/bootstrap.min.css" rel="stylesheet" integrity="sha384-sRIl4kxILFvY47J16cr9ZwB07vP4J8+LH7qKQnuqkuIAvNWLzeN8tE5YBujZqJLB" crossorigin="anonymous">
  </head>
  <body>

    <nav class="navbar bg-body-tertiary" data-bs-theme="dark">
      <div class="container-fluid">
        <a class="navbar-brand" href="/">TritonTube</a>
        <!-- Upload button in navbar -> opens modal -->
        <button class="btn btn-outline-light" type="button" data-bs-toggle="modal" data-bs-target="#uploadModal">
          Upload Video
        </button>
      </div>
    </nav>

    <!-- Upload Modal -->
    <div class="modal fade" id="uploadModal" tabindex="-1" aria-labelledby="uploadModalLabel" aria-hidden="true">
      <div class="modal-dialog">
        <div class="modal-content">
          <div class="modal-header">
            <h5 class="modal-title" id="uploadModalLabel">Upload an Video</h5>
            <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
          </div>
          <form action="/upload" method="post" enctype="multipart/form-data">
            <div class="modal-body">
              <div class="mb-3">
                <label for="videoFile" class="form-label">Select file</label>
                <input class="form-control" type="file" id="videoFile" name="file" accept="video/*" required />
              </div>
            </div>
            <div class="modal-footer">
              <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Close</button>
              <button type="submit" class="btn btn-primary">Upload</button>
            </div>
          </form>
        </div>
      </div>
    </div>

    <div class="container mt-3">
      {{if .}}
      <div class="row row-cols-1 row-cols-sm-2 row-cols-md-3 g-4">
        {{range .}}
        <div class="col">
          <div class="card h-100">
            <div class="ratio ratio-16x9">
              <!-- Small DASH preview player; muted and autoplaying -->
              <video id="preview-{{.EscapedId}}" class="card-img-top" playsinline muted autoplay loop data-mpd="/content/{{.Id}}/manifest.mpd"></video>
            </div>
            <div class="card-body">
              <h5 class="card-title text-truncate">{{.Id}}</h5>
              <p class="card-text"><small class="text-muted">Uploaded: {{.UploadTime}}</small></p>
              <a href="/videos/{{.EscapedId}}" class="stretched-link"></a>
            </div>
          </div>
        </div>
        {{end}}
      </div>
      {{else}}
      <div class="alert alert-secondary">No videos uploaded yet.</div>
      {{end}}
    </div>

    <script src="https://cdn.dashjs.org/latest/dash.all.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.8/dist/js/bootstrap.bundle.min.js" integrity="sha384-FKyoEForCGlyvwx9Hj09JcYn3nv7wiPVlz7YYwJrWVcXK/BmnVDxM+D2scQbITxI" crossorigin="anonymous"></script>

    <script>
      // Initialize a small dash.js player for every preview video element that has a data-mpd attribute
      (function(){
        document.addEventListener('DOMContentLoaded', function(){
          var videos = document.querySelectorAll('video[data-mpd]');
          videos.forEach(function(v){
            try {
              var mpd = v.getAttribute('data-mpd');
              var player = dashjs.MediaPlayer().create();
              player.initialize(v, mpd, true);
              // ensure muted so autoplay works across browsers
              v.muted = true;
              // small safety: attempt to play, but ignore failures
              var p = v.play();
              if (p && p.then) p.catch(function(e){});
            } catch (e) {
              // If dash.js fails for a preview, leave the card without an active preview
              console.warn('preview init failed', e);
            }
          });
        });
      })();
    </script>
  </body>
</html>
`

const videoHTML = `
<!DOCTYPE html>
<html>
  <head>
    <meta charset="UTF-8" />
    <title>{{.Id}} - TritonTube</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.8/dist/css/bootstrap.min.css" rel="stylesheet" integrity="sha384-sRIl4kxILFvY47J16cr9ZwB07vP4J8+LH7qKQnuqkuIAvNWLzeN8tE5YBujZqJLB" crossorigin="anonymous">
    <script src="https://cdn.dashjs.org/latest/dash.all.min.js"></script>
    <style>
      /* Small local tweaks for the video page */
      .video-card { box-shadow: 0 6px 18px rgba(0,0,0,0.08); }
      .meta-label { font-weight: 600; }
    </style>
  </head>
  <body>
    <nav class="navbar bg-body-tertiary" data-bs-theme="dark">
      <div class="container-fluid">
        <a class="navbar-brand" href="/">TritonTube</a>
        <div>
          <a href="/" class="btn btn-outline-light">Back</a>
        </div>
      </div>
    </nav>

    <div class="container my-4">
      <div class="row g-4">
        <div class="col-12">
          <div class="card video-card">
            <div class="ratio ratio-16x9">
              <video id="dashPlayer" class="w-100 h-100" controls playsinline></video>
            </div>
            <div class="card-body">
              <h4 class="card-title mb-1">{{.Id}}</h4>
              <p class="text-muted mb-0">Uploaded at: {{.UploadedAt}}</p>
            </div>
          </div>
        </div>
      </div>
    </div>

    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.8/dist/js/bootstrap.bundle.min.js" integrity="sha384-FKyoEForCGlyvwx9Hj09JcYn3nv7wiPVlz7YYwJrWVcXK/BmnVDxM+D2scQbITxI" crossorigin="anonymous"></script>
    <script>
      (function(){
        document.addEventListener('DOMContentLoaded', function(){
          try {
            var url = "/content/{{.Id}}/manifest.mpd";
            var player = dashjs.MediaPlayer().create();
            player.initialize(document.querySelector("#dashPlayer"), url, false);
          } catch (e) {
            console.error('dash player init failed', e);
            var el = document.querySelector('#dashPlayer');
            if (el) {
              el.outerHTML = '<div class="alert alert-warning">Unable to initialize DASH player in this browser.</div>';
            }
          }
        });
      })();
    </script>
  </body>
</html>
`
