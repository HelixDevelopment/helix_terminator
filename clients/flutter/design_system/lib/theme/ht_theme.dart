import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import '../tokens/ht_tokens.dart';

/// HelixTerminator Theme Configuration
/// 
/// Provides [ThemeData] for both dark and light modes,
/// fully aligned with the OpenDesign tokens.
class HTTheme {
  HTTheme._();

  static ThemeData dark() => _buildTheme(Brightness.dark);
  static ThemeData light() => _buildTheme(Brightness.light);

  static ThemeData _buildTheme(Brightness brightness) {
    final isDark = brightness == Brightness.dark;
    final t = HTTokens.color;

    final colorScheme = ColorScheme(
      brightness: brightness,
      primary: t.brandPrimary,
      onPrimary: t.textInverse,
      primaryContainer: t.brandPrimaryHover,
      onPrimaryContainer: t.textInverse,
      secondary: t.brandSecondary,
      onSecondary: t.textInverse,
      secondaryContainer: t.brandSecondary.withOpacity(0.2),
      onSecondaryContainer: t.textInverse,
      tertiary: t.brandAccent,
      onTertiary: t.textInverse,
      tertiaryContainer: t.brandAccent.withOpacity(0.2),
      onTertiaryContainer: t.textInverse,
      error: t.brandDanger,
      onError: t.textInverse,
      errorContainer: t.brandDanger.withOpacity(0.2),
      onErrorContainer: t.textInverse,
      surface: isDark ? t.backgroundPrimary : t.backgroundPrimaryLight,
      onSurface: isDark ? t.textPrimary : t.textPrimaryLight,
      surfaceContainerHighest: isDark ? t.backgroundSecondary : t.backgroundSecondaryLight,
      onSurfaceVariant: isDark ? t.textSecondary : t.textSecondaryLight,
      outline: isDark ? t.borderDefault : t.borderDefaultLight,
      outlineVariant: isDark ? t.borderDefault.withOpacity(0.5) : t.borderDefaultLight.withOpacity(0.5),
      shadow: Colors.black,
      scrim: isDark ? t.backgroundOverlay : t.backgroundOverlayLight,
      inverseSurface: isDark ? t.backgroundPrimaryLight : t.backgroundPrimary,
      onInverseSurface: isDark ? t.textPrimaryLight : t.textPrimary,
      surfaceTint: t.brandPrimary,
    );

    final textTheme = _buildTextTheme(isDark);

    return ThemeData(
      useMaterial3: true,
      brightness: brightness,
      colorScheme: colorScheme,
      scaffoldBackgroundColor: colorScheme.surface,
      cardColor: isDark ? t.backgroundElevated : t.backgroundElevatedLight,
      dividerColor: isDark ? t.borderDefault : t.borderDefaultLight,
      textTheme: textTheme,
      fontFamily: HTTokens.typography.fontFamilySans,
      appBarTheme: AppBarTheme(
        backgroundColor: colorScheme.surface,
        foregroundColor: colorScheme.onSurface,
        elevation: 0,
        centerTitle: false,
        titleTextStyle: textTheme.titleLarge,
        systemOverlayStyle: isDark ? SystemUiOverlayStyle.light : SystemUiOverlayStyle.dark,
      ),
      cardTheme: CardTheme(
        color: isDark ? t.backgroundElevated : t.backgroundElevatedLight,
        elevation: 0,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(HTTokens.border.radiusLg),
          side: BorderSide(
            color: isDark ? t.borderDefault : t.borderDefaultLight,
            width: HTTokens.border.widthThin,
          ),
        ),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: isDark ? t.backgroundSecondary : t.backgroundSecondaryLight,
        contentPadding: const EdgeInsets.symmetric(
          horizontal: 12,
          vertical: 14,
        ),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(HTTokens.border.radiusMd),
          borderSide: BorderSide(
            color: isDark ? t.borderDefault : t.borderDefaultLight,
            width: HTTokens.border.widthThin,
          ),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(HTTokens.border.radiusMd),
          borderSide: BorderSide(
            color: isDark ? t.borderDefault : t.borderDefaultLight,
            width: HTTokens.border.widthThin,
          ),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(HTTokens.border.radiusMd),
          borderSide: BorderSide(
            color: t.borderFocused,
            width: HTTokens.border.widthMedium,
          ),
        ),
        errorBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(HTTokens.border.radiusMd),
          borderSide: BorderSide(
            color: t.borderError,
            width: HTTokens.border.widthMedium,
          ),
        ),
        hintStyle: textTheme.bodyMedium?.copyWith(
          color: isDark ? t.textTertiary : t.textTertiaryLight,
        ),
        labelStyle: textTheme.bodyMedium?.copyWith(
          color: isDark ? t.textSecondary : t.textSecondaryLight,
        ),
      ),
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          backgroundColor: t.brandPrimary,
          foregroundColor: t.textInverse,
          minimumSize: const Size(0, 40),
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(HTTokens.border.radiusMd),
          ),
          textStyle: textTheme.labelLarge?.copyWith(
            fontWeight: HTTokens.typography.weightSemibold,
          ),
          elevation: 0,
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          foregroundColor: isDark ? t.textPrimary : t.textPrimaryLight,
          minimumSize: const Size(0, 40),
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          side: BorderSide(
            color: isDark ? t.borderDefault : t.borderDefaultLight,
            width: HTTokens.border.widthThin,
          ),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(HTTokens.border.radiusMd),
          ),
          textStyle: textTheme.labelLarge,
        ),
      ),
      textButtonTheme: TextButtonThemeData(
        style: TextButton.styleFrom(
          foregroundColor: t.brandPrimary,
          minimumSize: const Size(0, 40),
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(HTTokens.border.radiusMd),
          ),
          textStyle: textTheme.labelLarge,
        ),
      ),
      iconButtonTheme: IconButtonThemeData(
        style: IconButton.styleFrom(
          foregroundColor: isDark ? t.textSecondary : t.textSecondaryLight,
          minimumSize: const Size(44, 44),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(HTTokens.border.radiusMd),
          ),
        ),
      ),
      floatingActionButtonTheme: FloatingActionButtonThemeData(
        backgroundColor: t.brandPrimary,
        foregroundColor: t.textInverse,
        elevation: 0,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(HTTokens.border.radiusMd),
        ),
      ),
      bottomNavigationBarTheme: BottomNavigationBarThemeData(
        backgroundColor: isDark ? t.backgroundElevated : t.backgroundElevatedLight,
        selectedItemColor: t.brandPrimary,
        unselectedItemColor: isDark ? t.textSecondary : t.textSecondaryLight,
        type: BottomNavigationBarType.fixed,
        elevation: 0,
        showSelectedLabels: true,
        showUnselectedLabels: true,
      ),
      navigationRailTheme: NavigationRailThemeData(
        backgroundColor: isDark ? t.backgroundSecondary : t.backgroundSecondaryLight,
        selectedIconTheme: IconThemeData(color: t.brandPrimary, size: 24),
        unselectedIconTheme: IconThemeData(
          color: isDark ? t.textSecondary : t.textSecondaryLight,
          size: 24,
        ),
        selectedLabelTextStyle: textTheme.labelMedium?.copyWith(
          color: t.brandPrimary,
          fontWeight: HTTokens.typography.weightSemibold,
        ),
        unselectedLabelTextStyle: textTheme.labelMedium?.copyWith(
          color: isDark ? t.textSecondary : t.textSecondaryLight,
        ),
        elevation: 0,
      ),
      dialogTheme: DialogTheme(
        backgroundColor: isDark ? t.backgroundElevated : t.backgroundElevatedLight,
        elevation: 0,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(HTTokens.border.radiusLg),
        ),
      ),
      snackBarTheme: SnackBarThemeData(
        backgroundColor: isDark ? t.backgroundElevated : t.backgroundElevatedLight,
        contentTextStyle: textTheme.bodyMedium,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(HTTokens.border.radiusMd),
        ),
        behavior: SnackBarBehavior.floating,
        elevation: 0,
      ),
      chipTheme: ChipThemeData(
        backgroundColor: isDark ? t.backgroundTertiary : t.backgroundTertiaryLight,
        selectedColor: t.brandPrimary.withOpacity(0.2),
        labelStyle: textTheme.labelSmall,
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 0),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(HTTokens.border.radiusFull),
        ),
        side: BorderSide.none,
      ),
      tooltipTheme: TooltipThemeData(
        decoration: BoxDecoration(
          color: isDark ? t.backgroundElevated : t.backgroundElevatedLight,
          borderRadius: BorderRadius.circular(HTTokens.border.radiusSm),
          border: Border.all(
            color: isDark ? t.borderDefault : t.borderDefaultLight,
          ),
        ),
        textStyle: textTheme.bodySmall,
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      ),
      scrollbarTheme: ScrollbarThemeData(
        thumbColor: WidgetStateProperty.all(isDark ? t.textTertiary : t.textTertiaryLight),
        trackColor: WidgetStateProperty.all(Colors.transparent),
        radius: const Radius.circular(8),
        thickness: WidgetStateProperty.all(4),
      ),
      progressIndicatorTheme: ProgressIndicatorThemeData(
        color: t.brandPrimary,
        linearTrackColor: isDark ? t.backgroundTertiary : t.backgroundTertiaryLight,
        circularTrackColor: isDark ? t.backgroundTertiary : t.backgroundTertiaryLight,
      ),
      switchTheme: SwitchThemeData(
        thumbColor: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) return t.brandPrimary;
          return isDark ? t.backgroundTertiary : t.backgroundTertiaryLight;
        }),
        trackColor: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) return t.brandPrimary.withOpacity(0.3);
          return isDark ? t.backgroundTertiary : t.backgroundTertiaryLight;
        }),
      ),
      checkboxTheme: CheckboxThemeData(
        fillColor: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) return t.brandPrimary;
          return Colors.transparent;
        }),
        checkColor: WidgetStateProperty.all(t.textInverse),
        side: BorderSide(
          color: isDark ? t.borderDefault : t.borderDefaultLight,
        ),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(HTTokens.border.radiusSm),
        ),
      ),
      listTileTheme: ListTileThemeData(
        tileColor: Colors.transparent,
        selectedTileColor: isDark ? t.backgroundTertiary : t.backgroundTertiaryLight,
        contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        minVerticalPadding: 8,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(HTTokens.border.radiusMd),
        ),
      ),
      dataTableTheme: DataTableThemeData(
        headingRowColor: WidgetStateProperty.all(isDark ? t.backgroundSecondary : t.backgroundSecondaryLight),
        dataRowColor: WidgetStateProperty.all(Colors.transparent),
        dividerThickness: HTTokens.border.widthThin,
        horizontalMargin: 16,
        columnSpacing: 16,
        headingTextStyle: textTheme.labelSmall?.copyWith(
          fontWeight: HTTokens.typography.weightSemibold,
          color: isDark ? t.textSecondary : t.textSecondaryLight,
          letterSpacing: 0.5,
        ),
        dataTextStyle: textTheme.bodyMedium,
      ),
      tabBarTheme: TabBarTheme(
        labelColor: t.brandPrimary,
        unselectedLabelColor: isDark ? t.textSecondary : t.textSecondaryLight,
        indicatorColor: t.brandPrimary,
        indicatorSize: TabBarIndicatorSize.tab,
        labelStyle: textTheme.labelLarge?.copyWith(fontWeight: HTTokens.typography.weightSemibold),
        unselectedLabelStyle: textTheme.labelLarge,
      ),
      splashFactory: NoSplash.splashFactory,
      highlightColor: Colors.transparent,
    );
  }

  static TextTheme _buildTextTheme(bool isDark) {
    final t = HTTokens.color;
    final ty = HTTokens.typography;

    final Color primaryText = isDark ? t.textPrimary : t.textPrimaryLight;
    final Color secondaryText = isDark ? t.textSecondary : t.textSecondaryLight;

    return TextTheme(
      displayLarge: TextStyle(
        fontSize: ty.size5xl,
        fontWeight: ty.weightBold,
        height: ty.lineHeightTight,
        letterSpacing: -0.02,
        color: primaryText,
      ),
      displayMedium: TextStyle(
        fontSize: ty.size4xl,
        fontWeight: ty.weightBold,
        height: ty.lineHeightTight,
        letterSpacing: -0.02,
        color: primaryText,
      ),
      displaySmall: TextStyle(
        fontSize: ty.size3xl,
        fontWeight: ty.weightSemibold,
        height: ty.lineHeightTight,
        letterSpacing: -0.02,
        color: primaryText,
      ),
      headlineLarge: TextStyle(
        fontSize: ty.size2xl,
        fontWeight: ty.weightSemibold,
        height: ty.lineHeightTight,
        letterSpacing: -0.01,
        color: primaryText,
      ),
      headlineMedium: TextStyle(
        fontSize: ty.sizeXl,
        fontWeight: ty.weightSemibold,
        height: ty.lineHeightTight,
        letterSpacing: -0.01,
        color: primaryText,
      ),
      headlineSmall: TextStyle(
        fontSize: ty.sizeLg,
        fontWeight: ty.weightSemibold,
        height: 1.4,
        color: primaryText,
      ),
      titleLarge: TextStyle(
        fontSize: ty.sizeLg,
        fontWeight: ty.weightSemibold,
        height: 1.4,
        color: primaryText,
      ),
      titleMedium: TextStyle(
        fontSize: ty.sizeBase,
        fontWeight: ty.weightMedium,
        height: ty.lineHeightNormal,
        color: primaryText,
      ),
      titleSmall: TextStyle(
        fontSize: ty.sizeSm,
        fontWeight: ty.weightMedium,
        height: ty.lineHeightNormal,
        color: secondaryText,
      ),
      bodyLarge: TextStyle(
        fontSize: ty.sizeMd,
        fontWeight: ty.weightRegular,
        height: ty.lineHeightNormal,
        color: primaryText,
      ),
      bodyMedium: TextStyle(
        fontSize: ty.sizeBase,
        fontWeight: ty.weightRegular,
        height: ty.lineHeightNormal,
        color: primaryText,
      ),
      bodySmall: TextStyle(
        fontSize: ty.sizeSm,
        fontWeight: ty.weightRegular,
        height: ty.lineHeightNormal,
        color: secondaryText,
      ),
      labelLarge: TextStyle(
        fontSize: ty.sizeBase,
        fontWeight: ty.weightSemibold,
        height: ty.lineHeightNormal,
        color: primaryText,
      ),
      labelMedium: TextStyle(
        fontSize: ty.sizeSm,
        fontWeight: ty.weightMedium,
        height: ty.lineHeightNormal,
        color: secondaryText,
      ),
      labelSmall: TextStyle(
        fontSize: ty.sizeXs,
        fontWeight: ty.weightSemibold,
        height: 1.4,
        letterSpacing: 0.5,
        color: secondaryText,
      ),
    );
  }
}
