"use strict";
var pubs = require('../pubs.json');

exports.index = function(req, res) {
  res.render('index', {
    title: 'Joel H. W. Weinberger -- jww',
    extracss: [
      '/css/generic/basic-page.css',
      '/css/generic/header.css',
      '/css/page/index.css'
    ],
    header: 'jww (at) joelweinberger (dot) us',
    nohomelink: true
  });
};

exports.ajaxAbstracts = function(pubType) {
  return function(req, res) {
    res.render('abstract', {
      layout: false,
      nosectionheading: true,
      abstract: pubs[pubType][req.params[0]]
    });
  };
};

exports.ajaxBibtexs = function(pubType) {
  return function(req, res) {
    res.render('bibtex', {
      layout: false,
      nosectionheading: true,
      bibtex: pubs[pubType][req.params[0]]
    });
  };
};

function pubError(res, type) {
  res.render('pubError', {
    title: 'Joel H. W. Weinberger -- Unknown Publication',
    extracss: [
      '/css/generic/basic-page.css',
      '/css/generic/header.css'
    ],
    header: type,
    type: type
  });
}

exports.abstracts = function(pubType) {
  return function(req, res) {
    var pub = pubs[pubType][req.params[0]];
    if (!pub) {
      pubError(res, 'abstract');
      return;
    }

    res.render('abstract', {
      title: 'Joel H. W. Weinberger -- Paper Abstract',
      extracss: [
        '/css/generic/basic-page.css',
        '/css/generic/header.css'
      ],
      header: 'abstract',
      abstract: pub
    });
  };
};

exports.bibtexs = function(pubType) {
  return function(req, res) {
    var pub = pubs[pubType][req.params[0]];
    if (!pub) {
      pubError(res, 'bibtex');
      return;
    }

    res.render('bibtex', {
      title: 'Joel H. W. Weinberger -- Paper BibTeX',
      extracss: [
        '/css/generic/basic-page.css',
        '/css/generic/header.css'
      ],
      header: 'bibtex',
      bibtex: pub
    });
  };
};

exports.calendar = function(req, res) {
  res.render('calendar', {
    title: 'Joel H. W. Weinberger -- Calendar',
    extracss: [
      '/css/page/calendar.css'
    ],
    nocontent: true,
    nohomelink: true
  });
};

exports.publications = function(req, res) {
  res.render('publications', {
    title: 'Joel H. W. Weinberger -- Publications',
    extracss: [
      '/css/generic/basic-page.css',
      '/css/generic/header.css',
      '/css/page/index.css'
    ],
    extrascripts: [
      '/lib/jquery.min.js',
      '/js/index.js'
    ],
    header: 'publications',
    papers: pubs.papers,
    techs: pubs.techs
  });
};

exports.wedding = function(req, res) {
  res.render('wedding', {
    title: 'Joel H. W. Weinberger -- Wedding',
    extracss: [
      '/css/generic/basic-page.css',
      '/css/generic/header.css',
      '/css/page/index.css'
    ],
    header: 'wedding',
  });
};

// For legacy reasons (namely, the original blog), we need to redirect links
// frotm the original blog path to the new blog path so that old permalinks
// still work.
exports.blog = function(req, res) {
  var path = '/';
  if (req.params.length > 0) {
    path += req.params[0];
  }
  res.redirect('http://blog.joelweinberger.us' + path);
};
