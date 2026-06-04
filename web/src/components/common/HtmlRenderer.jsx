import React from 'react';
import DOMPurify from 'dompurify';

/**
 * 安全 HTML 渲染组件
 * 使用 DOMPurify 过滤 XSS 后渲染 HTML 内容
 */
const HtmlRenderer = ({ content, className = '', style = {} }) => {
  if (!content) return null;

  const sanitized = DOMPurify.sanitize(content, {
    ALLOWED_TAGS: [
      'p', 'br', 'div', 'span', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
      'strong', 'b', 'em', 'i', 'u', 's', 'strike', 'del',
      'a', 'img', 'video', 'audio', 'source',
      'ul', 'ol', 'li',
      'blockquote', 'pre', 'code',
      'table', 'thead', 'tbody', 'tr', 'th', 'td',
      'hr', 'sup', 'sub',
    ],
    ALLOWED_ATTR: [
      'href', 'target', 'rel', 'src', 'alt', 'title', 'width', 'height',
      'style', 'class', 'id', 'controls', 'loop', 'muted', 'autoplay',
      'poster', 'type', 'colspan', 'rowspan',
    ],
    ALLOW_DATA_ATTR: false,
  });

  return (
    <div
      className={`html-content ${className}`}
      style={style}
      dangerouslySetInnerHTML={{ __html: sanitized }}
    />
  );
};

export default HtmlRenderer;
