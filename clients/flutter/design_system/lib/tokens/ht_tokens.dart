import 'package:flutter/material.dart';

/// HelixTerminator Design System Tokens
/// 
/// All design values are derived from the OpenDesign tokens at:
/// submodules/open-design/design-systems/helixterminator/tokens.json
/// 
/// Usage:
/// ```dart
/// final color = HTTokens.color.brandPrimary;
/// final spacing = HTTokens.spacing.s4;
/// ```
class HTTokens {
  HTTokens._();

  // ── Colors ───────────────────────────────────────────────
  static const _ColorTokens color = _ColorTokens();
  static const _TypographyTokens typography = _TypographyTokens();
  static const _SpacingTokens spacing = _SpacingTokens();
  static const _ShadowTokens shadow = _ShadowTokens();
  static const _BorderTokens border = _BorderTokens();
  static const _SizeTokens size = _SizeTokens();
  static const _AnimationTokens animation = _AnimationTokens();
  static const _ZIndexTokens zIndex = _ZIndexTokens();
}

class _ColorTokens {
  const _ColorTokens();

  // Brand
  final Color brandPrimary = const Color(0xFF0A84FF);
  final Color brandPrimaryHover = const Color(0xFF0066CC);
  final Color brandPrimaryActive = const Color(0xFF0052A3);
  final Color brandPrimaryDisabled = const Color(0x660A84FF);
  final Color brandSecondary = const Color(0xFF5E5CE6);
  final Color brandAccent = const Color(0xFF30D158);
  final Color brandDanger = const Color(0xFFFF453A);
  final Color brandDangerHover = const Color(0xFFD92B20);
  final Color brandWarning = const Color(0xFFFF9F0A);
  final Color brandInfo = const Color(0xFF64D2FF);

  // Background (Dark)
  final Color backgroundPrimary = const Color(0xFF0D0D0D);
  final Color backgroundSecondary = const Color(0xFF1C1C1E);
  final Color backgroundTertiary = const Color(0xFF2C2C2E);
  final Color backgroundElevated = const Color(0xFF1C1C1E);
  final Color backgroundOverlay = const Color(0x99000000);

  // Background (Light)
  final Color backgroundPrimaryLight = const Color(0xFFFFFFFF);
  final Color backgroundSecondaryLight = const Color(0xFFF2F2F7);
  final Color backgroundTertiaryLight = const Color(0xFFE5E5EA);
  final Color backgroundElevatedLight = const Color(0xFFFFFFFF);
  final Color backgroundOverlayLight = const Color(0x66000000);

  // Text (Dark)
  final Color textPrimary = const Color(0xFFFFFFFF);
  final Color textSecondary = const Color(0x99EBEBF5);
  final Color textTertiary = const Color(0x4DEBEBF5);
  final Color textInverse = const Color(0xFF000000);

  // Text (Light)
  final Color textPrimaryLight = const Color(0xFF000000);
  final Color textSecondaryLight = const Color(0x993C3C43);
  final Color textTertiaryLight = const Color(0x4D3C3C43);
  final Color textInverseLight = const Color(0xFFFFFFFF);

  // Semantic Text
  final Color textBrand = const Color(0xFF0A84FF);
  final Color textDanger = const Color(0xFFFF453A);
  final Color textWarning = const Color(0xFFFF9F0A);
  final Color textSuccess = const Color(0xFF30D158);
  final Color textMonospace = const Color(0xFF30D158);
  final Color textMonospaceLight = const Color(0xFF248A3D);

  // Border
  final Color borderDefault = const Color(0xFF38383A);
  final Color borderDefaultLight = const Color(0xFFC6C6C8);
  final Color borderFocused = const Color(0xFF0A84FF);
  final Color borderError = const Color(0xFFFF453A);
  final Color borderSuccess = const Color(0xFF30D158);

  // Terminal ANSI
  final Color terminalBackground = const Color(0xFF0D0D0D);
  final Color terminalCursor = const Color(0xFF30D158);
  final Color terminalSelection = const Color(0x660A84FF);
  final Color terminalBlack = const Color(0xFF1C1C1E);
  final Color terminalRed = const Color(0xFFFF453A);
  final Color terminalGreen = const Color(0xFF30D158);
  final Color terminalYellow = const Color(0xFFFFD60A);
  final Color terminalBlue = const Color(0xFF0A84FF);
  final Color terminalMagenta = const Color(0xFFFF375F);
  final Color terminalCyan = const Color(0xFF64D2FF);
  final Color terminalWhite = const Color(0xFFFFFFFF);
  final Color terminalBrightBlack = const Color(0xFF48484A);
  final Color terminalBrightRed = const Color(0xFFFF6B6B);
  final Color terminalBrightGreen = const Color(0xFF5DE086);
  final Color terminalBrightYellow = const Color(0xFFFFE66D);
  final Color terminalBrightBlue = const Color(0xFF4DA3FF);
  final Color terminalBrightMagenta = const Color(0xFFFF6B8A);
  final Color terminalBrightCyan = const Color(0xFF8EE3FF);
  final Color terminalBrightWhite = const Color(0xFFF2F2F7);
}

class _TypographyTokens {
  const _TypographyTokens();

  final String fontFamilySans = 'Inter';
  final String fontFamilyMono = 'JetBrains Mono';

  final double sizeXs = 10.0;
  final double sizeSm = 12.0;
  final double sizeBase = 14.0;
  final double sizeMd = 16.0;
  final double sizeLg = 18.0;
  final double sizeXl = 20.0;
  final double size2xl = 24.0;
  final double size3xl = 30.0;
  final double size4xl = 36.0;
  final double size5xl = 48.0;

  final FontWeight weightRegular = FontWeight.w400;
  final FontWeight weightMedium = FontWeight.w500;
  final FontWeight weightSemibold = FontWeight.w600;
  final FontWeight weightBold = FontWeight.w700;

  final double lineHeightTight = 1.2;
  final double lineHeightNormal = 1.5;
  final double lineHeightRelaxed = 1.75;
}

class _SpacingTokens {
  const _SpacingTokens();

  final double s0 = 0.0;
  final double s1 = 4.0;
  final double s2 = 8.0;
  final double s3 = 12.0;
  final double s4 = 16.0;
  final double s5 = 20.0;
  final double s6 = 24.0;
  final double s8 = 32.0;
  final double s10 = 40.0;
  final double s12 = 48.0;
  final double s16 = 64.0;
  final double s20 = 80.0;
  final double s24 = 96.0;
}

class _ShadowTokens {
  const _ShadowTokens();

  final List<BoxShadow> none = const [];
  final List<BoxShadow> sm = const [
    BoxShadow(color: Color(0x4D000000), blurRadius: 2, offset: Offset(0, 1)),
  ];
  final List<BoxShadow> md = const [
    BoxShadow(color: Color(0x66000000), blurRadius: 6, offset: Offset(0, 4)),
    BoxShadow(color: Color(0x33000000), blurRadius: 4, offset: Offset(0, 2)),
  ];
  final List<BoxShadow> lg = const [
    BoxShadow(color: Color(0x80000000), blurRadius: 15, offset: Offset(0, 10)),
    BoxShadow(color: Color(0x4D000000), blurRadius: 6, offset: Offset(0, 4)),
  ];
  final List<BoxShadow> xl = const [
    BoxShadow(color: Color(0x99000000), blurRadius: 25, offset: Offset(0, 20)),
    BoxShadow(color: Color(0x66000000), blurRadius: 10, offset: Offset(0, 10)),
  ];
  final List<BoxShadow> glow = const [
    BoxShadow(color: Color(0x4D0A84FF), blurRadius: 15),
  ];
  final List<BoxShadow> glowDanger = const [
    BoxShadow(color: Color(0x4DFF453A), blurRadius: 15),
  ];
}

class _BorderTokens {
  const _BorderTokens();

  final double radiusNone = 0.0;
  final double radiusSm = 4.0;
  final double radiusMd = 8.0;
  final double radiusLg = 12.0;
  final double radiusXl = 16.0;
  final double radius2xl = 24.0;
  final double radiusFull = 9999.0;

  final double widthNone = 0.0;
  final double widthThin = 1.0;
  final double widthMedium = 2.0;
  final double widthThick = 4.0;
}

class _SizeTokens {
  const _SizeTokens();

  final double iconSm = 16.0;
  final double iconMd = 24.0;
  final double iconLg = 32.0;
  final double iconXl = 48.0;

  final double touchMin = 44.0;
  final double touchComfortable = 48.0;

  final double sidebarCollapsed = 64.0;
  final double sidebarExpanded = 240.0;

  final double appBarMobile = 56.0;
  final double appBarDesktop = 64.0;
}

class _AnimationTokens {
  const _AnimationTokens();

  final Duration fast = const Duration(milliseconds: 150);
  final Duration normal = const Duration(milliseconds: 250);
  final Duration slow = const Duration(milliseconds: 350);

  final Curve defaultCurve = const Cubic(0.4, 0, 0.2, 1);
  final Curve easeIn = const Cubic(0.4, 0, 1, 1);
  final Curve easeOut = const Cubic(0, 0, 0.2, 1);
  final Curve bounce = const Cubic(0.34, 1.56, 0.64, 1);
}

class _ZIndexTokens {
  const _ZIndexTokens();

  final double base = 0.0;
  final double dropdown = 100.0;
  final double sticky = 200.0;
  final double modal = 300.0;
  final double popover = 400.0;
  final double tooltip = 500.0;
  final double toast = 600.0;
}
