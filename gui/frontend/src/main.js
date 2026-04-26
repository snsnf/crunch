import './style.css';
import {SelectFiles, Compress, RevealFile, DownloadFFmpeg, HasGhostscript, SelectOutputDir, GetVersion} from '../wailsjs/go/main/App';
import {EventsOn, BrowserOpenURL} from '../wailsjs/runtime/runtime';

let selectedFiles = [];
let currentPreset = 'whatsapp';
let currentBitrate = 128;
let currentPdfQuality = 'ebook';
let isCompressing = false;
let outputDir = '';

// ─── FFmpeg setup ───

EventsOn('ffmpeg:missing', () => {
    showScreen('screen-setup');
});

EventsOn('ffmpeg:progress', (pct) => {
    const bar = document.getElementById('setup-bar');
    const label = document.getElementById('setup-pct');
    if (bar) bar.style.width = Math.round(pct) + '%';
    if (label) label.textContent = 'Downloading... ' + Math.round(pct) + '%';
});

EventsOn('ffmpeg:ready', () => {
    showScreen('screen-files');
});

EventsOn('ffmpeg:error', (msg) => {
    const text = document.getElementById('setup-text');
    text.textContent = 'Download failed: ' + msg;
    text.className = 'setup-error';
    const btn = document.getElementById('btn-setup');
    btn.disabled = false;
    btn.textContent = 'Retry';
    document.getElementById('setup-progress').style.display = 'none';
});

document.getElementById('btn-setup').addEventListener('click', () => {
    const btn = document.getElementById('btn-setup');
    btn.disabled = true;
    btn.textContent = 'Downloading...';
    document.getElementById('setup-progress').style.display = '';
    document.getElementById('setup-text').textContent = 'Downloading ffmpeg...';
    document.getElementById('setup-text').className = 'setup-text';
    DownloadFFmpeg();
});

// ─── Screen management ───

function showScreen(id) {
    document.querySelectorAll('.screen').forEach(s => s.classList.remove('active'));
    document.getElementById(id).classList.add('active');
}

// ─── DOM helper ───

function el(tag, attrs = {}, children = []) {
    const node = document.createElement(tag);
    for (const [k, v] of Object.entries(attrs)) {
        if (k === 'textContent') node.textContent = v;
        else if (k === 'className') node.className = v;
        else if (k.startsWith('on')) node.addEventListener(k.slice(2), v);
        else node.setAttribute(k, v);
    }
    for (const child of children) {
        if (typeof child === 'string') node.appendChild(document.createTextNode(child));
        else if (child) node.appendChild(child);
    }
    return node;
}

// ─── Error toast ───

function showError(message) {
    const existing = document.querySelector('.error-toast');
    if (existing) existing.remove();

    const toast = el('div', {className: 'error-toast', textContent: message});
    document.getElementById('app').appendChild(toast);
    setTimeout(() => toast.remove(), 3000);
}

// ─── File type detection ───

function dominantFileType() {
    const counts = {video: 0, image: 0, audio: 0, pdf: 0};
    selectedFiles.forEach(f => {
        if (counts[f.fileType] !== undefined) counts[f.fileType]++;
    });
    let best = 'video';
    let bestCount = 0;
    for (const [type, count] of Object.entries(counts)) {
        if (count > bestCount) { best = type; bestCount = count; }
    }
    return best;
}

// ─── File Selection ───

const dropZone = document.getElementById('drop-zone');
const fileList = document.getElementById('file-list');
const filesContent = document.getElementById('files-content');
const fileFooter = document.getElementById('file-footer');

async function openFilePicker() {
    try {
        const files = await SelectFiles();
        if (files && files.length > 0) {
            selectedFiles = files;
            renderFiles();
        }
    } catch (_) {
        // User cancelled
    }
}

dropZone.addEventListener('click', openFilePicker);
dropZone.addEventListener('keydown', (e) => {
    if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        openFilePicker();
    }
});

dropZone.addEventListener('dragover', (e) => {
    e.preventDefault();
    dropZone.classList.add('dragover');
});

dropZone.addEventListener('dragleave', () => {
    dropZone.classList.remove('dragover');
});

dropZone.addEventListener('drop', (e) => {
    e.preventDefault();
    dropZone.classList.remove('dragover');
});

EventsOn('files:dropped', (files) => {
    if (files && files.length > 0) {
        if (selectedFiles.length === 0) {
            selectedFiles = files;
        } else {
            selectedFiles = [...selectedFiles, ...files];
        }
        renderFiles();
    }
});

function renderFiles() {
    fileList.replaceChildren();
    if (selectedFiles.length === 0) {
        dropZone.style.display = '';
        filesContent.classList.remove('files-content--list');
        fileFooter.hidden = true;
        fileFooter.replaceChildren();
        return;
    }

    dropZone.style.display = 'none';
    filesContent.classList.add('files-content--list');
    fileFooter.hidden = false;

    selectedFiles.forEach((f, i) => {
        const metaParts = [];
        if (f.sizeMB > 0) metaParts.push(f.sizeMB + ' MB');
        else if (f.sizeKB > 0) metaParts.push(f.sizeKB + ' KB');
        if (f.width > 0) metaParts.push(f.width + '\u00d7' + f.height);
        if (f.duration > 0) metaParts.push(formatDuration(f.duration));

        const removeBtn = el('button', {
            className: 'file-remove',
            textContent: '\u00d7',
            'aria-label': 'Remove ' + f.name,
            onclick: () => {
                selectedFiles.splice(i, 1);
                renderFiles();
            },
        });

        const badge = el('span', {className: 'file-type-badge', textContent: f.fileType || '?'});

        fileList.appendChild(el('div', {className: 'file-item', role: 'listitem'}, [
            badge,
            el('span', {className: 'file-name', textContent: f.name}),
            el('span', {className: 'file-meta', textContent: metaParts.join('  \u00b7  ')}),
            removeBtn,
        ]));
    });

    fileFooter.replaceChildren(el('div', {className: 'file-actions'}, [
        el('button', {
            className: 'btn-ghost',
            textContent: 'Add more',
            onclick: async () => {
                try {
                    const files = await SelectFiles();
                    if (files && files.length > 0) {
                        selectedFiles = [...selectedFiles, ...files];
                        renderFiles();
                    }
                } catch (_) {}
            },
        }),
        el('button', {
            className: 'btn-primary btn-primary--compact',
            textContent: 'Next',
            onclick: goToSettings,
        }),
    ]));
}

function formatDuration(sec) {
    const m = Math.floor(sec / 60);
    const s = Math.floor(sec % 60);
    return m + ':' + s.toString().padStart(2, '0');
}

function parseSizeToBytes(s) {
    const n = parseFloat(s);
    if (s.endsWith('MB')) return n * 1024 * 1024;
    if (s.endsWith('KB')) return n * 1024;
    return n;
}

// ─── Settings ───

function getFileTypes() {
    const types = new Set();
    selectedFiles.forEach(f => { if (f.fileType) types.add(f.fileType); });
    return types;
}

function goToSettings() {
    const info = document.getElementById('selected-info');
    const count = selectedFiles.length;
    const types = getFileTypes();
    const typeList = [...types].join(', ');
    info.textContent = count + ' file' + (count > 1 ? 's' : '') + ' (' + typeList + ')';

    // Show settings for ALL present file types
    document.querySelectorAll('.settings-group').forEach(g => g.style.display = 'none');
    types.forEach(type => {
        const group = document.getElementById('settings-' + type);
        if (group) group.style.display = '';
    });

    // Warn if PDFs selected but Ghostscript not installed
    if (types.has('pdf')) {
        HasGhostscript().then(has => {
            if (!has) {
                showError('PDF support requires Ghostscript. Install from ghostscript.com');
            }
        });
    }

    showScreen('screen-settings');
}

// Video preset controls
document.querySelectorAll('[data-preset]').forEach(btn => {
    btn.addEventListener('click', () => {
        document.querySelectorAll('[data-preset]').forEach(b => {
            b.classList.remove('active');
            b.setAttribute('aria-checked', 'false');
        });
        btn.classList.add('active');
        btn.setAttribute('aria-checked', 'true');
        currentPreset = btn.dataset.preset;
        if (currentPreset === 'whatsapp') {
            document.getElementById('target-mb').value = 8;
        }
    });
});

// Audio bitrate controls
document.querySelectorAll('[data-bitrate]').forEach(btn => {
    btn.addEventListener('click', () => {
        document.querySelectorAll('[data-bitrate]').forEach(b => {
            b.classList.remove('active');
            b.setAttribute('aria-checked', 'false');
        });
        btn.classList.add('active');
        btn.setAttribute('aria-checked', 'true');
        currentBitrate = parseInt(btn.dataset.bitrate);
    });
});

// PDF quality controls
document.querySelectorAll('[data-pdfq]').forEach(btn => {
    btn.addEventListener('click', () => {
        document.querySelectorAll('[data-pdfq]').forEach(b => {
            b.classList.remove('active');
            b.setAttribute('aria-checked', 'false');
        });
        btn.classList.add('active');
        btn.setAttribute('aria-checked', 'true');
        currentPdfQuality = btn.dataset.pdfq;
    });
});

// Image quality slider
const imageQualitySlider = document.getElementById('image-quality');
const imageQualityVal = document.getElementById('image-quality-val');
if (imageQualitySlider) {
    imageQualitySlider.addEventListener('input', () => {
        imageQualityVal.textContent = imageQualitySlider.value;
    });
}

document.getElementById('btn-back').addEventListener('click', () => {
    showScreen('screen-files');
});

document.getElementById('btn-output-dir').addEventListener('click', async () => {
    try {
        const dir = await SelectOutputDir();
        if (dir) {
            outputDir = dir;
            const parts = dir.split('/');
            document.getElementById('output-dir-label').textContent = parts[parts.length - 1] || dir;
        }
    } catch (_) {}
});

document.getElementById('btn-compress').addEventListener('click', startCompression);

// ─── Compression ───

async function startCompression() {
    if (isCompressing) return;
    isCompressing = true;

    const compressBtn = document.getElementById('btn-compress');
    compressBtn.disabled = true;
    compressBtn.textContent = 'Compressing...';

    showScreen('screen-progress');
    document.getElementById('progress-bar').style.width = '0%';
    document.getElementById('progress-text').textContent = '0';
    document.getElementById('progress-track').setAttribute('aria-valuenow', '0');

    const opts = {
        files: selectedFiles.map(f => f.path),
        outputDir: outputDir,
        // Video
        preset: currentPreset,
        targetMB: parseInt(document.getElementById('target-mb').value) || 8,
        // Image
        imageQuality: parseInt(imageQualitySlider.value) || 75,
        // Audio
        audioBitrate: currentBitrate,
        // PDF
        pdfQuality: currentPdfQuality,
    };

    try {
        const results = await Compress(opts);
        document.getElementById('progress-bar').style.width = '100%';
        document.getElementById('progress-text').textContent = '100';
        document.getElementById('progress-track').setAttribute('aria-valuenow', '100');
        await new Promise(r => setTimeout(r, 350));
        showResults(results);
    } catch (e) {
        showError('Compression failed. Please try again.');
        showScreen('screen-settings');
    } finally {
        isCompressing = false;
        compressBtn.disabled = false;
        compressBtn.textContent = 'Compress';
    }
}

EventsOn('compress:file', (data) => {
    const label = document.getElementById('progress-file');
    label.textContent = data.name + '  (' + (data.index + 1) + ' of ' + data.total + ')';
    delete label.dataset.indeterminate;
    delete label.dataset.originalText;
});

EventsOn('compress:progress', (data) => {
    const overall = Math.round((data.index * 100 + data.percent) / data.total);
    const bar = document.getElementById('progress-bar');
    const text = document.getElementById('progress-text');
    const track = document.getElementById('progress-track');

    if (overall === 0) {
        // Unknown duration — show sweep animation so user knows work is happening
        track.classList.add('progress-track--indeterminate');
        text.classList.add('progress-pct--indeterminate');
        text.textContent = '0';
        track.removeAttribute('aria-valuenow');
        document.getElementById('progress-file').dataset.indeterminate = '1';
        const label = document.getElementById('progress-file');
        if (!label.dataset.originalText) label.dataset.originalText = label.textContent;
        label.textContent = label.dataset.originalText + '  ·  This may take a minute...';
    } else {
        // Real progress — switch back to fill bar
        track.classList.remove('progress-track--indeterminate');
        text.classList.remove('progress-pct--indeterminate');
        bar.style.width = overall + '%';
        text.textContent = overall;
        track.setAttribute('aria-valuenow', String(overall));
        const label = document.getElementById('progress-file');
        if (label.dataset.indeterminate) {
            label.textContent = label.dataset.originalText || label.textContent;
            delete label.dataset.indeterminate;
            delete label.dataset.originalText;
        }
    }
});

// ─── Results ───

function showResults(results) {
    showScreen('screen-results');

    const list = document.getElementById('results-list');
    list.replaceChildren();

    let hasErrors = false;

    results.forEach(r => {
        const item = el('div', {className: 'result-item', role: 'listitem'});

        if (r.error) {
            hasErrors = true;
            item.appendChild(el('div', {className: 'result-name', textContent: basename(r.inputPath)}));
            item.appendChild(el('div', {className: 'result-error', textContent: r.error}));
        } else {
            item.appendChild(el('div', {className: 'result-name', textContent: basename(r.outputPath)}));

            const statItems = [
                el('span', {className: 'result-stat', textContent: r.inputSize + ' \u2192 ' + r.outputSize}),
            ];
            if (r.inputRes && r.outputRes) {
                statItems.push(el('span', {className: 'result-stat', textContent: r.inputRes + ' \u2192 ' + r.outputRes}));
            }

            // Calculate saved percentage \u2014 normalize both sizes to the same unit first
            const inBytes = parseSizeToBytes(r.inputSize);
            const outBytes = parseSizeToBytes(r.outputSize);
            if (inBytes > 0 && outBytes > 0) {
                const saved = Math.round((1 - outBytes / inBytes) * 100);
                if (saved > 0) {
                    statItems.push(el('span', {className: 'result-stat result-stat--saved', textContent: '\u2212' + saved + '%'}));
                }
            }

            item.appendChild(el('div', {className: 'result-stats'}, statItems));

            if (r.note) {
                item.appendChild(el('div', {className: 'result-note', textContent: r.note}));
            }

            item.appendChild(el('button', {
                className: 'result-open',
                textContent: 'Show in Finder \u2192',
                'aria-label': 'Reveal ' + basename(r.outputPath) + ' in Finder',
                onclick: () => RevealFile(r.outputPath),
            }));
        }

        list.appendChild(item);
    });

    if (hasErrors) {
        document.querySelector('#screen-results .screen-title').textContent = 'Done (with errors)';
    } else {
        document.querySelector('#screen-results .screen-title').textContent = 'Done';
    }
}

function basename(path) {
    return path.split('/').pop().split('\\').pop();
}

// ─── About modal ───

const aboutBackdrop = document.getElementById('about-backdrop');
const aboutModal = document.getElementById('about-modal');

GetVersion().then(v => {
    document.getElementById('about-version').textContent = 'v' + v;
});

document.getElementById('btn-about').addEventListener('click', () => {
    aboutBackdrop.hidden = false;
    document.getElementById('about-close').focus();
});

function closeAbout() {
    aboutBackdrop.hidden = true;
}

document.getElementById('about-close').addEventListener('click', closeAbout);

document.getElementById('about-github').addEventListener('click', () => {
    BrowserOpenURL('https://github.com/snsnf');
});

aboutBackdrop.addEventListener('click', (e) => {
    if (!aboutModal.contains(e.target)) closeAbout();
});

document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && !aboutBackdrop.hidden) closeAbout();
});

document.getElementById('btn-new').addEventListener('click', () => {
    selectedFiles = [];
    outputDir = '';
    document.getElementById('output-dir-label').textContent = 'Same folder as original';
    fileList.replaceChildren();
    dropZone.style.display = '';
    filesContent.classList.remove('files-content--list');
    fileFooter.hidden = true;
    fileFooter.replaceChildren();
    showScreen('screen-files');
});
