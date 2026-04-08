/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React from 'react';
import { Popover } from '@douyinfe/semi-ui';
import { QrCode, Sparkles } from 'lucide-react';

const GroupQRCodeButton = ({ qrCodeUrl, qrCodeLink, isMobile, t }) => {
  if (!qrCodeUrl) {
    return null;
  }

  const targetUrl = qrCodeLink || qrCodeUrl;

  const handleOpenQRCode = () => {
    window.open(targetUrl, '_blank', 'noopener,noreferrer');
  };

  const content = (
    <div className='group-qr-popover'>
      <div className='group-qr-popover-title'>{t('扫码加入交流群')}</div>
      <a
        href={targetUrl}
        target='_blank'
        rel='noreferrer'
        className='group-qr-popover-image-link'
      >
        <img
          src={qrCodeUrl}
          alt={t('交流群二维码')}
          className='group-qr-popover-image'
          loading='lazy'
        />
      </a>
      <div className='group-qr-popover-hint'>
        {t('悬停自动展开，点击二维码可查看原图')}
      </div>
    </div>
  );

  return (
    <Popover
      content={content}
      position='bottom'
      trigger={isMobile ? 'click' : 'hover'}
      showArrow
      spacing={12}
    >
      <button
        type='button'
        onClick={handleOpenQRCode}
        className='group-qr-entry'
        aria-label={t('加入交流群')}
      >
        <Sparkles size={14} className='group-qr-entry-sparkle' />
        <QrCode size={16} />
        <span>{t('加入交流群')}</span>
      </button>
    </Popover>
  );
};

export default GroupQRCodeButton;
