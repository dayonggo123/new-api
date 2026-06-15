/*
 * Prompt 业务场景标签派生规则
 * 基于现有 tags/标题/描述/内容 自动映射出业务场景，零后端改动
 */

export const SCENE_RULES = [
  {
    scene: '电商',
    icon: '🛒',
    color: 'orange',
    keywords: ['电商', 'e-commerce', 'ecommerce', 'shop', '产品图', '产品照', '商品', 'listing', 'product photo', 'white background', 'clean background', 'studio shot', '商品图', '淘宝', '天猫', 'amazon', 'shein', 'temu'],
  },
  {
    scene: '短视频',
    icon: '📱',
    color: 'pink',
    keywords: ['短视频', 'short video', 'tiktok', 'reels', 'shorts', '抖音', '快手', '竖屏', 'vertical video', 'vlog'],
  },
  {
    scene: 'TVC',
    icon: '📺',
    color: 'red',
    keywords: ['tvc', '电视广告', 'commercial', 'brand film', '广告片', 'tvc', '产品视频', '广告'],
  },
  {
    scene: '广告',
    icon: '📢',
    color: 'red',
    keywords: ['advertisement', 'advertising', 'campaign', 'poster ad', 'banner', '户外广告', '信息流', 'feed ad'],
  },
  {
    scene: '纪录片',
    icon: '🎬',
    color: 'blue',
    keywords: ['纪录片', 'documentary', 'interview', '访谈', '纪实', 'candid', 'real life'],
  },
  {
    scene: '教学',
    icon: '🎓',
    color: 'green',
    keywords: ['教学', 'tutorial', 'education', 'course', '教程', 'how-to', '培训', '课件', '微课', 'mooc'],
  },
  {
    scene: '社交媒体',
    icon: '💬',
    color: 'lime',
    keywords: ['社交媒体', 'social media', 'instagram', 'facebook', 'twitter', '小红书', '微博', '朋友圈', '种草', 'post'],
  },
  {
    scene: '品牌设计',
    icon: '🎨',
    color: 'gold',
    keywords: ['品牌', 'branding', 'brand identity', 'logo', 'vi', 'visual identity', '海报', 'poster', '品牌视觉'],
  },
  {
    scene: '个人IP',
    icon: '⭐',
    color: 'purple',
    keywords: ['个人ip', 'personal brand', 'influencer', '头像', 'profile picture', '形象照', '自我展示', 'portrait'],
  },
  {
    scene: '自媒体',
    icon: '📡',
    color: 'cyan',
    keywords: ['自媒体', 'self-media', '内容创作', '博主', 'up主', '创作者', 'content creator'],
  },
  {
    scene: '直播',
    icon: '🔴',
    color: 'red',
    keywords: ['直播', 'live stream', 'livestream', '直播间', '主播', 'live commerce'],
  },
  {
    scene: '出版印刷',
    icon: '📖',
    color: 'grey',
    keywords: ['出版', 'print', 'printing', '杂志', 'magazine', '书籍', 'book', '画册', 'brochure'],
  },
  {
    scene: '游戏CG',
    icon: '🎮',
    color: 'violet',
    keywords: ['游戏', 'game', 'cg', 'character design', 'concept art', 'unreal', 'unity', '游戏角色', '场景设计'],
  },
  {
    scene: '影视后期',
    icon: '🎞️',
    color: 'indigo',
    keywords: ['影视后期', 'vfx', 'visual effects', '特效', '合成', 'compositing', '电影', 'film', 'cinema'],
  },
  {
    scene: '动画制作',
    icon: '🐾',
    color: 'magenta',
    keywords: ['动画', 'animation', 'anime', 'cartoon', 'motion graphics', 'mg动画', 'motion design'],
  },
  {
    scene: '产品摄影',
    icon: '📷',
    color: 'orange',
    keywords: ['产品摄影', 'product photography', '静物', 'still life', '商品摄影', 'catalog'],
  },
  {
    scene: '人像写真',
    icon: '👤',
    color: 'purple',
    keywords: ['人像', 'portrait', '写真', 'headshot', 'model', '人物', 'face', 'beauty'],
  },
  {
    scene: '风景摄影',
    icon: '🏞️',
    color: 'teal',
    keywords: ['风景', 'landscape', 'nature', '风光', '旅行', 'travel', 'mountain', 'ocean', 'sunset', '航拍'],
  },
  {
    scene: '美食',
    icon: '🍜',
    color: 'amber',
    keywords: ['美食', 'food', 'cuisine', 'dish', 'restaurant', 'menu', '食物摄影', 'food photography'],
  },
  {
    scene: '时尚',
    icon: '👗',
    color: 'fuchsia',
    keywords: ['时尚', 'fashion', 'runway', 'couture', '街拍', 'streetwear', 'lookbook', 'model fashion'],
  },
];

/**
 * 从提示词数据派生业务场景标签
 * @param {Object} prompt - prompt 对象，至少包含 tags/title/description/content
 * @returns {Array<{scene: string, icon: string, color: string}>} 匹配到的场景列表
 */
export function deriveScenes(prompt) {
  if (!prompt) return [];

  const rawTags = prompt.tags;
  let tagList = [];
  if (typeof rawTags === 'string' && rawTags.trim()) {
    try {
      tagList = JSON.parse(rawTags);
    } catch (e) {
      tagList = rawTags.split(',').map((t) => t.trim()).filter(Boolean);
    }
  } else if (Array.isArray(rawTags)) {
    tagList = rawTags;
  }

  const haystack = [
    prompt.title,
    prompt.description,
    prompt.content,
    prompt.content_en,
    ...tagList,
  ]
    .filter(Boolean)
    .join(' ')
    .toLowerCase();

  const matched = [];
  for (const rule of SCENE_RULES) {
    const hit = rule.keywords.some((kw) => haystack.includes(kw.toLowerCase()));
    if (hit) {
      matched.push({ scene: rule.scene, icon: rule.icon, color: rule.color });
    }
  }

  return matched;
}

/**
 * 获取所有可用的业务场景列表（用于筛选器）
 */
export function getSceneOptions() {
  return SCENE_RULES.map((r) => ({
    label: `${r.icon} ${r.scene}`,
    value: r.scene,
    color: r.color,
  }));
}

/**
 * 判断某个提示词是否匹配指定场景
 */
export function matchScene(prompt, sceneName) {
  if (!sceneName) return true;
  return deriveScenes(prompt).some((s) => s.scene === sceneName);
}
