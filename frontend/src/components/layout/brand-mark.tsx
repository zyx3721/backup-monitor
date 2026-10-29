import { useId } from 'react';
import { defaultBrandSettings } from '@/lib/branding';

type BrandMarkProps = {
  className?: string;
  label?: string;
};

// BrandMark 以内联 SVG 渲染默认品牌图标，可被基础配置上传的图标覆盖
export function BrandMark({ className, label }: BrandMarkProps) {
  const uid = useId().replace(/:/g, '');
  const tileId = `bm-logo-tile-${uid}`;

  return (
    <svg
      viewBox="0 0 512 512"
      className={className}
      role={label ? 'img' : undefined}
      aria-label={label}
      aria-hidden={label ? undefined : true}
      focusable="false"
    >
      <defs>
        <linearGradient
          id={tileId}
          x1="85.3333"
          y1="53.3333"
          x2="437.3333"
          y2="469.3333"
          gradientUnits="userSpaceOnUse"
        >
          <stop offset="0" stopColor="#3b82f6" />
          <stop offset="0.52" stopColor="#06b6d4" />
          <stop offset="1" stopColor="#10b981" />
        </linearGradient>
      </defs>
      <rect width="512" height="512" rx="112" fill={`url(#${tileId})`} />
      <path
        d="M256 92 L396 146 V266 C396 346 336 404 256 428 C176 404 116 346 116 266 V146 Z"
        fill="#ffffff"
        fillOpacity={0.14}
        stroke="#ffffff"
        strokeWidth={22}
        strokeLinejoin="round"
      />
      <path
        d="M168 264 H214 L240 210 L288 310 L312 264 H348"
        fill="none"
        stroke="#ffffff"
        strokeWidth={26}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

type BrandIconProps = {
  iconData: string;
  alt: string;
  className?: string;
};

// BrandIcon 按品牌配置渲染图标：默认图标走内联 BrandMark，自定义图标回退为 img
export function BrandIcon({ iconData, alt, className }: BrandIconProps) {
  if (iconData === defaultBrandSettings.iconData) {
    return <BrandMark className={className} label={alt} />;
  }
  return <img src={iconData} alt={alt} className={className} />;
}
